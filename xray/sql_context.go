// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
)

func registerDriver() {
	for _, d := range sql.Drivers() {
		db, err := sql.Open(d, "")
		if err != nil {
			continue
		}
		sql.Register(d+":xray", &driverDriver{
			Driver: db.Driver(),
		})
		db.Close()
	}
}

var registerOnce sync.Once

func initXRayDriver() {
	registerOnce.Do(registerDriver)
}

// SQLContext opens a normalized and traced wrapper around an *sql.DB connection.
// It uses `sql.Open` internally and shares the same function signature.
// To ensure passwords are filtered, it is HIGHLY RECOMMENDED that your DSN
// follows the format: `<schema>://<user>:<password>@<host>:<port>/<database>`
func SQLContext(driver, dsn string) (*sql.DB, error) {
	initXRayDriver()
	return sql.Open(driver+":xray", dsn)
}

type driverDriver struct {
	driver.Driver
}

func (driver *driverDriver) Open(dsn string) (driver.Conn, error) {
	rawConn, err := driver.Open(dsn)
	if err != nil {
		return nil, err
	}

	conn := &driverConn{Conn: rawConn}

	// Detect if DSN is a URL or not, set appropriate attribute
	urlDsn := dsn
	if !strings.Contains(dsn, "//") {
		urlDsn = "//" + urlDsn
	}
	// Here we're trying to detect things like `host:port/database` as a URL, which is pretty hard
	// So we just assume that if it's got a scheme, a user, or a query that it's probably a URL
	if u, err := url.Parse(urlDsn); err == nil && (u.Scheme != "" || u.User != nil || u.RawQuery != "" || strings.Contains(u.Path, "@")) {
		// Check that this isn't in the form of user/pass@host:port/db, as that will shove the host into the path
		if strings.Contains(u.Path, "@") {
			u, err = url.Parse(fmt.Sprintf("%s//%s%%2F%s", u.Scheme, u.Host, u.Path[1:]))
			if err != nil {
				return nil, err
			}
		}

		// Strip password from user:password pair in address
		if u.User != nil {
			uname := u.User.Username()

			// Some drivers use "user/pass@host:port" instead of "user:pass@host:port"
			// So we must manually attempt to chop off a potential password.
			// But we can skip this if we already found the password.
			if _, ok := u.User.Password(); !ok {
				uname = strings.Split(uname, "/")[0]
			}

			u.User = url.User(uname)
		}

		// Strip password from query parameters
		q := u.Query()
		q.Del("password")
		u.RawQuery = q.Encode()

		conn.url = u.String()
		if !strings.Contains(dsn, "//") {
			conn.url = conn.url[2:]
		}
	} else {
		// We don't *think* it's a URL, so now we have to try our best to strip passwords from
		// some unknown DSL. We attempt to detect whether it's space-delimited or semicolon-delimited
		// then remove any keys with the name "password" or "pwd". This won't catch everything, but
		// from surveying the current (Jan 2017) landscape of drivers it should catch most.
		conn.connectionString = stripPasswords(dsn)
	}

	// TODO: Detect database type and use that to populate attributes
	conn.databaseType = "Unknown"
	conn.databaseVersion = "Unknown"
	conn.user = "Unknown"
	conn.dbname = "Unknown"

	// There's no standard to get SQL driver version information
	// So we invent an interface by which drivers can provide us this data
	type versionedDriver interface {
		Version() string
	}

	if vd, ok := driver.Driver.(versionedDriver); ok {
		conn.driverVersion = vd.Version()
	} else {
		t := reflect.TypeOf(driver.Driver)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		conn.driverVersion = t.PkgPath()
	}

	return conn, nil
}

type driverConn struct {
	driver.Conn

	connectionString string
	url              string
	databaseType     string
	databaseVersion  string
	driverVersion    string
	user             string
	dbname           string
}

func (conn *driverConn) Ping(ctx context.Context) error {
	return Capture(ctx, conn.dbname, func(ctx context.Context) error {
		conn.populate(ctx, "PING")
		if p, ok := conn.Conn.(driver.Pinger); ok {
			return p.Ping(ctx)
		}
		return nil
	})
}

func (conn *driverConn) Prepare(query string) (driver.Stmt, error) {
	panic("not supported")
}

func (conn *driverConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	var stmt driver.Stmt
	var err error
	if connCtx, ok := conn.Conn.(driver.ConnPrepareContext); ok {
		stmt, err = connCtx.PrepareContext(ctx, query)
	} else {
		stmt, err = conn.Conn.Prepare(query)
		if err == nil {
			select {
			default:
			case <-ctx.Done():
				stmt.Close()
				return nil, ctx.Err()
			}
		}
	}
	if err != nil {
		return nil, err
	}
	return &driverStmt{
		Stmt:  stmt,
		conn:  conn,
		query: query,
	}, nil
}

func (conn *driverConn) Begin() (driver.Tx, error) {
	panic("not supported")
}

func (conn *driverConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	var tx driver.Tx
	var err error
	if connCtx, ok := conn.Conn.(driver.ConnBeginTx); ok {
		tx, err = connCtx.BeginTx(ctx, opts)
	} else {
		if opts.Isolation != driver.IsolationLevel(sql.LevelDefault) {
			return nil, errors.New("xray: driver does not support non-default isolation level")
		}
		if opts.ReadOnly {
			return nil, errors.New("xray: driver does not support read-only transactions")
		}
		tx, err = conn.Conn.Begin()
		if err == nil {
			select {
			default:
			case <-ctx.Done():
				tx.Rollback()
				return nil, ctx.Err()
			}
		}
	}
	if err != nil {
		return nil, err
	}
	return &driverTx{Tx: tx}, nil
}

func (conn *driverConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	panic("not supported")
}

func (conn *driverConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := conn.Conn.(driver.Execer)
	if !ok {
		return nil, driver.ErrSkip
	}

	var err error
	var result driver.Result
	if execerCtx, ok := conn.Conn.(driver.ExecerContext); ok {
		err = Capture(ctx, conn.dbname, func(ctx context.Context) error {
			conn.populate(ctx, query)
			var err error
			result, err = execerCtx.ExecContext(ctx, query, args)
			return err
		})
	} else {
		select {
		default:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		dargs, err0 := namedValuesToValues(args)
		if err0 != nil {
			return nil, err0
		}
		err = Capture(ctx, conn.dbname, func(ctx context.Context) error {
			conn.populate(ctx, query)
			var err error
			result, err = execer.Exec(query, dargs)
			return err
		})
	}
	return result, err
}

func (conn *driverConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	panic("not supported")
}

func (conn *driverConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := conn.Conn.(driver.Queryer)
	if !ok {
		return nil, driver.ErrSkip
	}

	var err error
	var rows driver.Rows
	if queryerCtx, ok := conn.Conn.(driver.QueryerContext); ok {
		err = Capture(ctx, conn.dbname, func(ctx context.Context) error {
			conn.populate(ctx, query)
			var err error
			rows, err = queryerCtx.QueryContext(ctx, query, args)
			return err
		})
	} else {
		select {
		default:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		dargs, err0 := namedValuesToValues(args)
		if err0 != nil {
			return nil, err0
		}
		err = Capture(ctx, conn.dbname, func(ctx context.Context) error {
			conn.populate(ctx, query)
			var err error
			rows, err = queryer.Query(query, dargs)
			return err
		})
	}
	return rows, err
}

func (conn *driverConn) Close() error {
	return conn.Conn.Close()
}

func (conn *driverConn) populate(ctx context.Context, query string) {
	seg := GetSegment(ctx)

	if seg == nil {
		processNilSegment(ctx)
		return
	}

	seg.Lock()
	seg.Namespace = "remote"
	seg.GetSQL().ConnectionString = conn.connectionString
	seg.GetSQL().URL = conn.url
	seg.GetSQL().DatabaseType = conn.databaseType
	seg.GetSQL().DatabaseVersion = conn.databaseVersion
	seg.GetSQL().DriverVersion = conn.driverVersion
	seg.GetSQL().User = conn.user
	seg.GetSQL().SanitizedQuery = query
	seg.Unlock()
}

type driverTx struct {
	driver.Tx
}

func (tx *driverTx) Commit() error {
	return tx.Tx.Commit()
}

func (tx *driverTx) Rollback() error {
	return tx.Tx.Rollback()
}

type driverStmt struct {
	driver.Stmt
	conn  *driverConn
	query string
}

func (stmt *driverStmt) Close() error {
	return stmt.Stmt.Close()
}

func (stmt *driverStmt) NumInput() int {
	return stmt.Stmt.NumInput()
}

func (stmt *driverStmt) Exec(args []driver.Value) (driver.Result, error) {
	panic("not supported")
}

func (stmt *driverStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	var result driver.Result
	var err error
	if execerContext, ok := stmt.Stmt.(driver.StmtExecContext); ok {
		err = Capture(ctx, stmt.conn.dbname, func(ctx context.Context) error {
			stmt.conn.populate(ctx, stmt.query)
			var err error
			result, err = execerContext.ExecContext(ctx, args)
			return err
		})
	} else {
		select {
		default:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		dargs, err0 := namedValuesToValues(args)
		if err0 != nil {
			return nil, err0
		}
		err = Capture(ctx, stmt.conn.dbname, func(ctx context.Context) error {
			stmt.conn.populate(ctx, stmt.query)
			var err error
			result, err = stmt.Stmt.Exec(dargs)
			return err
		})
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (stmt *driverStmt) Query(args []driver.Value) (driver.Rows, error) {
	panic("not supported")
}

func (stmt *driverStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	var result driver.Rows
	var err error
	if queryCtx, ok := stmt.Stmt.(driver.StmtQueryContext); ok {
		err = Capture(ctx, stmt.conn.dbname, func(ctx context.Context) error {
			stmt.conn.populate(ctx, stmt.query)
			var err error
			result, err = queryCtx.QueryContext(ctx, args)
			return err
		})
	} else {
		select {
		default:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		dargs, err0 := namedValuesToValues(args)
		if err0 != nil {
			return nil, err0
		}
		err = Capture(ctx, stmt.conn.dbname, func(ctx context.Context) error {
			stmt.conn.populate(ctx, stmt.query)
			var err error
			result, err = stmt.Stmt.Query(dargs)
			return err
		})
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (stmt *driverStmt) ColumnConverter(idx int) driver.ValueConverter {
	if conv, ok := stmt.Stmt.(driver.ColumnConverter); ok {
		return conv.ColumnConverter(idx)
	}
	return driver.DefaultParameterConverter
}

func namedValuesToValues(args []driver.NamedValue) ([]driver.Value, error) {
	var err error
	ret := make([]driver.Value, len(args))
	for _, arg := range args {
		if len(arg.Name) > 0 {
			err = errors.New("xray: driver does not support the use of Named Parameters")
		}
		ret[arg.Ordinal-1] = arg.Value
	}
	return ret, err
}

func stripPasswords(dsn string) string {
	var (
		tok        bytes.Buffer
		res        bytes.Buffer
		isPassword bool
		inBraces   bool
		delimiter  byte = ' '
	)
	flush := func() {
		if inBraces {
			return
		}
		if !isPassword {
			res.Write(tok.Bytes())
		}
		tok.Reset()
		isPassword = false
	}
	if strings.Count(dsn, ";") > strings.Count(dsn, " ") {
		delimiter = ';'
	}

	buf := strings.NewReader(dsn)
	for c, err := buf.ReadByte(); err == nil; c, err = buf.ReadByte() {
		tok.WriteByte(c)
		switch c {
		case ':', delimiter:
			flush()
		case '=':
			tokStr := strings.ToLower(tok.String())
			isPassword = `password=` == tokStr || `pwd=` == tokStr
			if b, err := buf.ReadByte(); err == nil && b == '{' {
				inBraces = true
			}
			buf.UnreadByte()
		case '}':
			b, err := buf.ReadByte()
			if err != nil {
				break
			}
			if b == '}' {
				tok.WriteByte(b)
			} else {
				inBraces = false
				buf.UnreadByte()
			}
		}
	}
	inBraces = false
	flush()
	return res.String()
}

func processNilSegment(ctx context.Context) {
	cfg := GetRecorder(ctx)
	failedMessage := "failed to get segment from context since segment is nil"
	if cfg != nil && cfg.ContextMissingStrategy != nil {
		cfg.ContextMissingStrategy.ContextMissing(failedMessage)
	} else {
		globalCfg.ContextMissingStrategy().ContextMissing(failedMessage)
	}
}

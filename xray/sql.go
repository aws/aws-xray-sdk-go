// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"context"
	"database/sql"
)

// SQL opens a normalized and traced wrapper around an *sql.DB connection.
// It uses `sql.Open` internally and shares the same function signature.
// To ensure passwords are filtered, it is HIGHLY RECOMMENDED that your DSN
// follows the format: `<schema>://<user>:<password>@<host>:<port>/<database>`
//
// Deprecated: SQL exists for historical compatibility.
// Use SQLContext insted of SQL, it can be used
// as a drop-in replacement of the database/sql package.
func SQL(driver, dsn string) (*DB, error) {
	db, err := SQLContext(driver, dsn)
	if err != nil {
		return nil, err
	}
	return &DB{DB: db}, nil
}

// DB copies the interface of sql.DB but adds X-Ray tracing.
// It must be created with xray.SQL.
type DB struct {
	*sql.DB
}

// Begin starts a transaction.
func (db *DB) Begin(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

// Prepare creates a prepared statement for later queries or executions.
func (db *DB) Prepare(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &Stmt{Stmt: stmt}, nil
}

// Exec captures executing a query without returning any rows and
// adds corresponding information into subsegment.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(ctx, query, args...)
}

// Query captures executing a query that returns rows and adds corresponding information into subsegment.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(ctx, query, args)
}

// QueryRow captures executing a query that is expected to return at most one row
// and adds corresponding information into subsegment.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(ctx, query, args...)
}

// Tx copies the interface of sql.Tx but adds X-Ray tracing.
// It must be created with xray.DB.Begin.
type Tx struct {
	*sql.Tx
}

// Prepare creates a prepared statement for later queries or executions.
func (tx *Tx) Prepare(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := tx.Tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &Stmt{Stmt: stmt}, err
}

// Exec captures executing a query that doesn't return rows and adds
// corresponding information into subsegment.
func (tx *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return tx.Tx.ExecContext(ctx, query, args...)
}

// Query captures executing a query that returns rows and adds corresponding information into subsegment.
func (tx *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.Tx.QueryContext(ctx, query, args...)
}

// QueryRow captures executing a query that is expected to return at most one row and adds
// corresponding information into subsegment.
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return tx.QueryRowContext(ctx, query, args...)
}

// Stmt returns a transaction-specific prepared statement from an existing statement.
func (tx *Tx) Stmt(ctx context.Context, stmt *Stmt) *Stmt {
	return &Stmt{
		Stmt: tx.StmtContext(ctx, stmt.Stmt),
	}
}

// Stmt copies the interface of sql.Stmt but adds X-Ray tracing.
// It must be created with xray.DB.Prepare or xray.Tx.Stmt.
type Stmt struct {
	*sql.Stmt
}

// Exec captures executing a prepared statement with the given arguments and
// returning a Result summarizing the effect of the statement and adds corresponding
// information into subsegment.
func (stmt *Stmt) Exec(ctx context.Context, args ...interface{}) (sql.Result, error) {
	return stmt.ExecContext(ctx, args...)
}

// Query captures executing a prepared query statement with the given arguments
// and returning the query results as a *Rows and adds corresponding information
// into subsegment.
func (stmt *Stmt) Query(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	return stmt.QueryContext(ctx, args...)
}

// QueryRow captures executing a prepared query statement with the given arguments and
// adds corresponding information into subsegment.
func (stmt *Stmt) QueryRow(ctx context.Context, args ...interface{}) *sql.Row {
	return stmt.QueryRowContext(ctx, args...)
}

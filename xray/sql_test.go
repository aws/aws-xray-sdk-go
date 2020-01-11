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
	"crypto/rand"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

func TestSQL(t *testing.T) {
	suite.Run(t, &sqlTestSuite{
		dbs: map[string]sqlmock.Sqlmock{},
	})
}

type sqlTestSuite struct {
	suite.Suite

	dbs map[string]sqlmock.Sqlmock

	dsn  string
	db   *sql.DB
	mock sqlmock.Sqlmock
}

func (s *sqlTestSuite) mockDB(dsn string) {
	if dsn == "" {
		b := make([]byte, 32)
		rand.Read(b)
		dsn = string(b)
	}

	var err error
	s.dsn = dsn
	if mock, ok := s.dbs[dsn]; ok {
		s.mock = mock
	} else {
		_, s.mock, err = sqlmock.NewWithDSN(dsn)
		s.Require().NoError(err)
		s.dbs[dsn] = s.mock
	}
}

func (s *sqlTestSuite) connect() {
	var err error
	s.db, err = SQLContext("sqlmock", s.dsn)
	s.Require().NoError(err)
}

func (s *sqlTestSuite) mockPSQL(err error) {
	row := sqlmock.NewRows([]string{"version()", "current_user", "current_database()"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	s.mock.ExpectPrepare(`SELECT version\(\), current_user, current_database\(\)`).ExpectQuery().WillReturnRows(row)
}
func (s *sqlTestSuite) mockMySQL(err error) {
	row := sqlmock.NewRows([]string{"version()", "current_user()", "database()"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	s.mock.ExpectPrepare(`SELECT version\(\), current_user\(\), database\(\)`).ExpectQuery().WillReturnRows(row)
}
func (s *sqlTestSuite) mockMSSQL(err error) {
	row := sqlmock.NewRows([]string{"@@version", "current_user", "db_name()"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	s.mock.ExpectPrepare(`SELECT @@version, current_user, db_name\(\)`).ExpectQuery().WillReturnRows(row)
}
func (s *sqlTestSuite) mockOracle(err error) {
	row := sqlmock.NewRows([]string{"version", "user", "ora_database_name"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	s.mock.ExpectPrepare(`SELECT version FROM v\$instance UNION SELECT user, ora_database_name FROM dual`).ExpectQuery().WillReturnRows(row)
}

func (s *sqlTestSuite) TestPasswordlessURL() {
	s.mockDB("postgres://user@host:5432/database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("postgres://user@host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPasswordURL() {
	s.mockDB("postgres://user:password@host:5432/database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("postgres://user@host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPasswordURLQuery() {
	s.mockDB("postgres://user:password@host:5432/database?password=password")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("postgres://user@host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPasswordURLSchemaless() {
	s.mockDB("user:password@host:5432/database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("user@host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPasswordURLSchemalessUserlessQuery() {
	s.mockDB("host:5432/database?password=password")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestWeirdPasswordURL() {
	s.mockDB("user%2Fpassword@host:5432/database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("user@host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestWeirderPasswordURL() {
	s.mockDB("user/password@host:5432/database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("", conn.attr.connectionString)
		s.Equal("user@host:5432/database", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPasswordlessConnectionString() {
	s.mockDB("user=user database=database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("user=user database=database", conn.attr.connectionString)
		s.Equal("", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPasswordConnectionString() {
	s.mockDB("user=user password=password database=database")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("user=user database=database", conn.attr.connectionString)
		s.Equal("", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)

	s.mockDB("")
	s.mockPSQL(nil)
	s.connect()
}

func (s *sqlTestSuite) TestSemicolonPasswordConnectionString() {
	s.mockDB("odbc:server=localhost;user id=sa;password={foo}};bar};otherthing=thing")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("odbc:server=localhost;user id=sa;otherthing=thing", conn.attr.connectionString)
		s.Equal("", conn.attr.url)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestPSQL() {
	s.mockDB("")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("Postgres", conn.attr.databaseType)
		s.Equal("test version", conn.attr.databaseVersion)
		s.Equal("test user", conn.attr.user)
		s.Equal("test database", conn.attr.dbname)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestMySQL() {
	s.mockDB("")
	s.mockPSQL(errors.New(""))
	s.mockMySQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("MySQL", conn.attr.databaseType)
		s.Equal("test version", conn.attr.databaseVersion)
		s.Equal("test user", conn.attr.user)
		s.Equal("test database", conn.attr.dbname)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestMSSQL() {
	s.mockDB("")
	s.mockPSQL(errors.New(""))
	s.mockMySQL(errors.New(""))
	s.mockMSSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("MS SQL", conn.attr.databaseType)
		s.Equal("test version", conn.attr.databaseVersion)
		s.Equal("test user", conn.attr.user)
		s.Equal("test database", conn.attr.dbname)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestOracle() {
	s.mockDB("")
	s.mockPSQL(errors.New(""))
	s.mockMySQL(errors.New(""))
	s.mockMSSQL(errors.New(""))
	s.mockOracle(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("Oracle", conn.attr.databaseType)
		s.Equal("test version", conn.attr.databaseVersion)
		s.Equal("test user", conn.attr.user)
		s.Equal("test database", conn.attr.dbname)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestUnknownDatabase() {
	s.mockDB("")
	s.mockPSQL(errors.New(""))
	s.mockMySQL(errors.New(""))
	s.mockMSSQL(errors.New(""))
	s.mockOracle(errors.New(""))
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Equal("Unknown", conn.attr.databaseType)
		s.Equal("Unknown", conn.attr.databaseVersion)
		s.Equal("Unknown", conn.attr.user)
		s.Equal("Unknown", conn.attr.dbname)
		return nil
	})
	s.Require().NoError(err)
}

func (s *sqlTestSuite) TestDriverVersionPackage() {
	s.mockDB("")
	s.mockPSQL(nil)
	s.connect()

	ctx, seg := BeginSegment(context.Background(), "test")
	defer seg.Close(nil)
	conn, err := s.db.Conn(ctx)
	s.Require().NoError(err)
	defer conn.Close()
	s.Require().NoError(s.mock.ExpectationsWereMet())
	err = conn.Raw(func(rawConn interface{}) error {
		conn, ok := rawConn.(*driverConn)
		if !ok {
			return errors.New("unexpected connection type")
		}
		s.Contains(conn.attr.driverVersion, "DATA-DOG/go-sqlmock")
		return nil
	})
}

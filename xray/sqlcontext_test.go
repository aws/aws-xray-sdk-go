// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// utility functions for testing SQL

func mockPostgreSQL(mock sqlmock.Sqlmock, err error) {
	row := sqlmock.NewRows([]string{"version()", "current_user", "current_database()"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	mock.ExpectPrepare(`SELECT version\(\), current_user, current_database\(\)`).ExpectQuery().WillReturnRows(row)
}
func mockMySQL(mock sqlmock.Sqlmock, err error) {
	row := sqlmock.NewRows([]string{"version()", "current_user()", "database()"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	mock.ExpectPrepare(`SELECT version\(\), current_user\(\), database\(\)`).ExpectQuery().WillReturnRows(row)
}
func mockMSSQL(mock sqlmock.Sqlmock, err error) {
	row := sqlmock.NewRows([]string{"@@version", "current_user", "db_name()"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	mock.ExpectPrepare(`SELECT @@version, current_user, db_name\(\)`).ExpectQuery().WillReturnRows(row)
}
func mockOracle(mock sqlmock.Sqlmock, err error) {
	row := sqlmock.NewRows([]string{"version", "user", "ora_database_name"}).
		AddRow("test version", "test user", "test database").
		RowError(0, err)
	mock.ExpectPrepare(`SELECT version FROM v\$instance UNION SELECT user, ora_database_name FROM dual`).ExpectQuery().WillReturnRows(row)
}

func capturePing(dsn string) (*Segment, error) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	db, err := SQLContext("sqlmock", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ctx, root := BeginSegment(ctx, "test")
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	root.Close(nil)

	seg, err := td.Recv()
	if err != nil {
		return nil, err
	}
	var subseg *Segment
	if err := json.Unmarshal(seg.Subsegments[0], &subseg); err != nil {
		return nil, err
	}

	return subseg, nil
}

func TestDSN(t *testing.T) {
	tc := []struct {
		dsn string
		url string
		str string
	}{
		{
			dsn: "postgres://user@host:5432/database",
			url: "postgres://user@host:5432/database",
		},
		{
			dsn: "postgres://user:password@host:5432/database",
			url: "postgres://user@host:5432/database",
		},
		{
			dsn: "postgres://host:5432/database?password=password",
			url: "postgres://host:5432/database",
		},
		{
			dsn: "user:password@host:5432/database",
			url: "user@host:5432/database",
		},
		{
			dsn: "host:5432/database?password=password",
			url: "host:5432/database",
		},
		{
			dsn: "user%2Fpassword@host:5432/database",
			url: "user@host:5432/database",
		},
		{
			dsn: "user/password@host:5432/database",
			url: "user@host:5432/database",
		},
		{
			dsn: "user=user database=database",
			str: "user=user database=database",
		},
		{
			dsn: "user=user password=password database=database",
			str: "user=user database=database",
		},
		{
			dsn: "odbc:server=localhost;user id=sa;password={foo}};bar};otherthing=thing",
			str: "odbc:server=localhost;user id=sa;otherthing=thing",
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.dsn, func(t *testing.T) {
			db, mock, err := sqlmock.NewWithDSN(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mockPostgreSQL(mock, nil)

			subseg, err := capturePing(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())

			assert.Equal(t, "remote", subseg.Namespace)
			assert.Equal(t, "Postgres", subseg.SQL.DatabaseType)
			assert.Equal(t, tt.url, subseg.SQL.URL)
			assert.Equal(t, tt.str, subseg.SQL.ConnectionString)
			assert.Equal(t, "test version", subseg.SQL.DatabaseVersion)
			assert.Equal(t, "test user", subseg.SQL.User)
			assert.False(t, subseg.Throttle)
			assert.False(t, subseg.Error)
			assert.False(t, subseg.Fault)
		})
	}
}

func TestPostgreSQL(t *testing.T) {
	dsn := "test-postgre"
	db, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mockPostgreSQL(mock, nil)

	subseg, err := capturePing(dsn)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "Postgres", subseg.SQL.DatabaseType)
	assert.Equal(t, "", subseg.SQL.URL)
	assert.Equal(t, dsn, subseg.SQL.ConnectionString)
	assert.Equal(t, "test version", subseg.SQL.DatabaseVersion)
	assert.Equal(t, "test user", subseg.SQL.User)
	assert.False(t, subseg.Throttle)
	assert.False(t, subseg.Error)
	assert.False(t, subseg.Fault)
}

func TestMySQL(t *testing.T) {
	dsn := "test-mysql"
	db, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mockPostgreSQL(mock, errors.New("syntax error"))
	mockMySQL(mock, nil)

	subseg, err := capturePing(dsn)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "MySQL", subseg.SQL.DatabaseType)
	assert.Equal(t, "", subseg.SQL.URL)
	assert.Equal(t, dsn, subseg.SQL.ConnectionString)
	assert.Equal(t, "test version", subseg.SQL.DatabaseVersion)
	assert.Equal(t, "test user", subseg.SQL.User)
	assert.False(t, subseg.Throttle)
	assert.False(t, subseg.Error)
	assert.False(t, subseg.Fault)
}

func TestMSSQL(t *testing.T) {
	dsn := "test-mssql"
	db, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mockPostgreSQL(mock, errors.New("syntax error"))
	mockMySQL(mock, errors.New("syntax error"))
	mockMSSQL(mock, nil)

	subseg, err := capturePing(dsn)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "MS SQL", subseg.SQL.DatabaseType)
	assert.Equal(t, "", subseg.SQL.URL)
	assert.Equal(t, dsn, subseg.SQL.ConnectionString)
	assert.Equal(t, "test version", subseg.SQL.DatabaseVersion)
	assert.Equal(t, "test user", subseg.SQL.User)
	assert.False(t, subseg.Throttle)
	assert.False(t, subseg.Error)
	assert.False(t, subseg.Fault)
}

func TestOracle(t *testing.T) {
	dsn := "test-oracle"
	db, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mockPostgreSQL(mock, errors.New("syntax error"))
	mockMySQL(mock, errors.New("syntax error"))
	mockMSSQL(mock, errors.New("syntax error"))
	mockOracle(mock, nil)

	subseg, err := capturePing(dsn)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "Oracle", subseg.SQL.DatabaseType)
	assert.Equal(t, "", subseg.SQL.URL)
	assert.Equal(t, dsn, subseg.SQL.ConnectionString)
	assert.Equal(t, "test version", subseg.SQL.DatabaseVersion)
	assert.Equal(t, "test user", subseg.SQL.User)
	assert.False(t, subseg.Throttle)
	assert.False(t, subseg.Error)
	assert.False(t, subseg.Fault)
}

func TestUnknownDatabase(t *testing.T) {
	dsn := "test-unknown"
	db, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mockPostgreSQL(mock, errors.New("syntax error"))
	mockMySQL(mock, errors.New("syntax error"))
	mockMSSQL(mock, errors.New("syntax error"))
	mockOracle(mock, errors.New("syntax error"))

	subseg, err := capturePing(dsn)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "Unknown", subseg.SQL.DatabaseType)
	assert.Equal(t, "", subseg.SQL.URL)
	assert.Equal(t, dsn, subseg.SQL.ConnectionString)
	assert.Equal(t, "Unknown", subseg.SQL.DatabaseVersion)
	assert.Equal(t, "Unknown", subseg.SQL.User)
	assert.False(t, subseg.Throttle)
	assert.False(t, subseg.Error)
	assert.False(t, subseg.Fault)
}

func TestStripPasswords(t *testing.T) {
	tc := []struct {
		in   string
		want string
	}{
		{
			in:   "user=user database=database",
			want: "user=user database=database",
		},
		{
			in:   "user=user password=password database=database",
			want: "user=user database=database",
		},
		{
			in:   "odbc:server=localhost;user id=sa;password={foo}};bar};otherthing=thing",
			want: "odbc:server=localhost;user id=sa;otherthing=thing",
		},

		// see https://github.com/aws/aws-xray-sdk-go/issues/181
		{
			in:   "password=",
			want: "",
		},
		{
			in:   "pwd=",
			want: "",
		},

		// test cases for https://github.com/go-sql-driver/mysql
		{
			in:   "user:password@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
			want: "user@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
		},

		{
			in:   "user@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
			want: "user@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
		},

		{
			in:   "user:password@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
			want: "user@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
		},

		{
			in:   "user@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
			want: "user@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
		},

		{
			in:   "user:password@/",
			want: "user@/",
		},
	}

	for _, tt := range tc {
		got := stripPasswords(tt.in)
		if got != tt.want {
			t.Errorf("%s: want %s, got %s", tt.in, tt.want, got)
		}
	}
}

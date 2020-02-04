// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// +build go1.10

package xray

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type versionedDriver struct {
	driver.Driver
	version string
}

func (d *versionedDriver) Version() string {
	return d.version
}

func TestDriverVersion(t *testing.T) {
	dsn := "test-versioned-driver"
	db, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mockPostgreSQL(mock, nil)

	// implement versionedDriver
	driver := &versionedDriver{
		Driver:  db.Driver(),
		version: "3.1415926535",
	}
	connector := &fallbackConnector{
		driver: driver,
		name:   dsn,
	}
	sqlConnector := SQLConnector("sanitized-dsn", connector)
	db = sql.OpenDB(sqlConnector)
	defer db.Close()

	ctx, td := NewTestDaemon()
	defer td.Close()

	// Execute SQL
	ctx, root := BeginSegment(ctx, "test")
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	root.Close(nil)
	assert.NoError(t, mock.ExpectationsWereMet())

	// assertion
	seg, err := td.Recv()
	if err != nil {
		t.Fatal(err)
	}
	var subseg *Segment
	if err := json.Unmarshal(seg.Subsegments[0], &subseg); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sanitized-dsn", subseg.SQL.ConnectionString)
	assert.Equal(t, "3.1415926535", subseg.SQL.DriverVersion)
}

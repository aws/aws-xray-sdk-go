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

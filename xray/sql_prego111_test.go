// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// +build !go1.11

package xray

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestMySQLPasswordConnectionString(t *testing.T) {
	tc := []struct {
		dsn string
		url string
		str string
	}{
		{
			dsn: "username:password@protocol(address:1234)/dbname?param=value",
			url: "username@protocol(address:1234)/dbname?param=value",
		},
		{
			dsn: "username@protocol(address:1234)/dbname?param=value",
			url: "username@protocol(address:1234)/dbname?param=value",
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
			mockMySQL(mock, nil)

			subseg, err := capturePing(tt.dsn)
			if err != nil {
				t.Fatal(err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())

			assert.Equal(t, "remote", subseg.Namespace)
			assert.Equal(t, "MySQL", subseg.SQL.DatabaseType)
			assert.Equal(t, tt.url, subseg.SQL.URL)
			assert.Equal(t, tt.str, subseg.SQL.ConnectionString)
		})
	}
}

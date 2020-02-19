// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// +build go1.11

package xray

func (s *sqlTestSuite) TestMySQLPasswordConnectionString() {
	s.mockDB("username:password@protocol(address:1234)/dbname?param=value")
	s.mockMySQL(nil)
	s.connect()

	s.Require().NoError(s.mock.ExpectationsWereMet())
	if s.db.connectionString != "" {
		s.Equal("username@protocol(address:1234)/dbname?param=value", s.db.connectionString)
		s.Equal("", s.db.url)
	}
	if s.db.url != "" {
		s.Equal("username@protocol(address:1234)/dbname?param=value", s.db.url)
		s.Equal("", s.db.connectionString)
	}
}

func (s *sqlTestSuite) TestMySQLPasswordlessConnectionString() {
	s.mockDB("username@protocol(address:1234)/dbname?param=value")
	s.mockMySQL(nil)
	s.connect()

	s.Require().NoError(s.mock.ExpectationsWereMet())
	if s.db.connectionString != "" {
		s.Equal("username@protocol(address:1234)/dbname?param=value", s.db.connectionString)
		s.Equal("", s.db.url)
	}
	if s.db.url != "" {
		s.Equal("username@protocol(address:1234)/dbname?param=value", s.db.url)
		s.Equal("", s.db.connectionString)
	}
}

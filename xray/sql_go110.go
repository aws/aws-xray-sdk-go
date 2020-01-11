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
	"context"
	"database/sql/driver"
)

func (conn *driverConn) ResetSession(ctx context.Context) error {
	if sr, ok := conn.Conn.(driver.SessionResetter); ok {
		return Capture(ctx, conn.dbname, func(ctx context.Context) error {
			conn.populate(ctx, "RESET SESSION")
			return sr.ResetSession(ctx)
		})
	}
	return nil
}

type driverConnector struct {
	driver.Connector
	driver *driverDriver
}

func (c *driverConnector) Connect(ctx context.Context) (driver.Conn, error) {
	var rawConn driver.Conn
	err := Capture(ctx, "TODO", func(ctx context.Context) error {
		var err error
		rawConn, err = c.Connector.Connect(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	conn := &driverConn{Conn: rawConn}

	return conn, nil
}

func (c *driverConnector) Driver() driver.Driver {
	return c.driver
}

type fallbackConnector struct {
	driver driver.Driver
	name   string
}

func (c *fallbackConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.driver.Open(c.name)
	if err != nil {
		return nil, err
	}
	select {
	default:
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	}
	return conn, nil
}

func (c *fallbackConnector) Driver() driver.Driver {
	return c.driver
}

func (d *driverDriver) OpenConnector(name string) (driver.Connector, error) {
	if dctx, ok := d.Driver.(driver.DriverContext); ok {
		c, err := dctx.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return &driverConnector{
			driver:    d,
			Connector: c,
		}, nil
	}
	return &driverConnector{
		driver: d,
		Connector: &fallbackConnector{
			driver: d.Driver,
			name:   name,
		},
	}, nil
}

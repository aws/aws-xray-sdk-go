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
		return Capture(ctx, conn.attr.dbname, func(ctx context.Context) error {
			conn.attr.populate(ctx, "RESET SESSION")
			return sr.ResetSession(ctx)
		})
	}
	return nil
}

type driverConnector struct {
	driver.Connector
	driver *driverDriver
	attr   *dbAttribute
}

func (c *driverConnector) Connect(ctx context.Context) (driver.Conn, error) {
	var rawConn driver.Conn
	err := Capture(ctx, c.attr.dbname, func(ctx context.Context) error {
		c.attr.populate(ctx, "CONNECT")
		var err error
		rawConn, err = c.Connector.Connect(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	conn := &driverConn{
		Conn: rawConn,
		attr: c.attr,
	}

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
	var c driver.Connector
	if dctx, ok := d.Driver.(driver.DriverContext); ok {
		var err error
		c, err = dctx.OpenConnector(name)
		if err != nil {
			return nil, err
		}
	} else {
		c = &fallbackConnector{
			driver: d.Driver,
			name:   name,
		}
	}
	attr, err := newDBAttribute(context.Background(), d.Driver, name)
	if err != nil {
		return nil, err
	}
	c = &driverConnector{
		Connector: c,
		driver:    d,
		attr:      attr,
	}
	return c, nil
}

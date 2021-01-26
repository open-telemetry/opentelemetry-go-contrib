// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelsql

import (
	"context"
	"database/sql/driver"
)

var _ driver.Connector = (*otConnector)(nil)

type otConnector struct {
	driver.Connector
	otDriver *otDriver
}

func newConnector(connector driver.Connector, otDriver *otDriver) *otConnector {
	return &otConnector{
		Connector: connector,
		otDriver:  otDriver,
	}
}

func (c *otConnector) Connect(ctx context.Context) (connection driver.Conn, err error) {
	connection, err = c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return newConn(connection, c.otDriver), nil
}

func (c *otConnector) Driver() driver.Driver {
	return c.otDriver
}

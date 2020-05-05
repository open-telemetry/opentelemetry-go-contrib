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

package driver

import (
	"context"

	"database/sql/driver"
)

type otelConnector struct {
	c driver.Connector
	d *tracingSetup
	s *tracingSetup
}

var _ driver.Connector = &otelConnector{}

func newConnector(realConnector driver.Connector, driverSetup, connSetup *tracingSetup) driver.Connector {
	return &otelConnector{
		c: realConnector,
		d: driverSetup,
		s: connSetup,
	}
}

// Connect is a part of an implementation of the driver.Connector
// interface. It forwards the call to the actual connector and puts
// the result in the tracing wrapper.
func (c *otelConnector) Connect(ctx context.Context) (driver.Conn, error) {
	ctx, span := c.s.StartNoStmt(ctx, "open")
	realConn, err := c.c.Connect(ctx)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	return maybeNewConn(realConn, c.s), err
}

// Driver is a part of an implementation of the driver.Connector
// interface. It forwards the call to the actual connector and puts
// the result in the tracing wrapper.
func (c *otelConnector) Driver() driver.Driver {
	return newDriver(c.c.Driver(), c.d)
}

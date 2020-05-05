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

package sql

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	tracedriver "go.opentelemetry.io/contrib/plugins/database/sql/driver"
)

// Register wraps the passed driver into the tracing driver, generates
// a driver name based on the passed one, and registers the driver
// under the generated name. The passed driver name should remain the
// same as it would be passed to the Register function from the
// standard library.
//
// Register will create a new tracing driver with no options. If
// customization of the driver is desired then create the tracing
// driver explicitly and pass it to sql.Register under a customized
// name:
// 	tracingDriver := tracedriver.NewDriver(
// 		realDriver,
// 		tracedriver.WithTracer(someTracer),
// 	)
// 	sql.Register("otel-foo", tracingDriver)
func Register(driverName string, realDriver driver.Driver) {
	otelName := otelDriverName(driverName)
	otelDriver := tracedriver.NewDriver(realDriver)
	sql.Register(otelName, otelDriver)
}

// Open opens a database using a generated driver name based on the
// passed name. The passed driver name should remain the same as it
// would be passed to the Open function from the standard library.
func Open(driverName, dataSourceName string) (*sql.DB, error) {
	otelName := otelDriverName(driverName)
	return sql.Open(otelName, dataSourceName)
}

func otelDriverName(driverName string) string {
	return fmt.Sprintf("otel-%s", driverName)
}

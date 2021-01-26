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
	"database/sql"
	"database/sql/driver"
	"errors"
	"strconv"
	"sync"
)

var registerLock sync.Mutex

// Register initializes and registers our OTel wrapped database driver
// identified by its driverName, using provided Option.
// It is possible to register multiple wrappers for the same database driver if
// needing different Option for different connections.
// Parameter dbSystem is an identifier for the database management system (DBMS)
// product being used.
//
// For more information, see semantic conventions for database
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/database.md
func Register(driverName string, dbSystem string, options ...Option) (string, error) {
	// Retrieve the driver implementation we need to wrap with instrumentation
	db, err := sql.Open(driverName, "")
	if err != nil {
		return "", err
	}
	dri := db.Driver()
	if err = db.Close(); err != nil {
		return "", err
	}

	registerLock.Lock()
	defer registerLock.Unlock()

	// Since we might want to register multiple OTel drivers to have different
	// configurations, but potentially the same underlying database driver, we
	// cycle through to find available driver names.
	driverName = driverName + "-otelsql-"
	for i := int64(0); i < 1000; i++ {
		var (
			found   = false
			regName = driverName + strconv.FormatInt(i, 10)
		)
		for _, name := range sql.Drivers() {
			if name == regName {
				found = true
			}
		}
		if !found {
			sql.Register(regName, newDriver(dri, newConfig(dbSystem, options...)))
			return regName, nil
		}
	}
	return "", errors.New("unable to register driver, all slots have been taken")
}

// WrapDriver takes a SQL driver and wraps it with OTel instrumentation.
// Parameter dbSystem is an identifier for the database management system (DBMS)
// product being used.
//
// For more information, see semantic conventions for database
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/database.md
func WrapDriver(dri driver.Driver, dbSystem string, options ...Option) driver.Driver {
	return newDriver(dri, newConfig(dbSystem, options...))
}

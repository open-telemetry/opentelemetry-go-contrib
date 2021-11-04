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

package otelsql_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/open-telemetry/opentelemetry-go-contrib/instrumentation/database/otelsql"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	_ "github.com/mattn/go-sqlite3"
)

func init() {
	otelsql.PostCall = func(ctx context.Context, err error, elapse time.Duration, attrs ...attribute.KeyValue) {
		span := trace.SpanFromContext(ctx)
		log.Printf("trace_id: %s,space_id: %s, %v cost: %s",
			span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String(), attrs, elapse)
	}
}

func ExampleDB_QueryContext() {
	{
		// Register our sqlite3-otel wrapper for the provided SQLite3 driver.
		// "sqlite3-otel" must not be registered, set in func init(){} as recommended.
		sql.Register("sqlite3-otel", otelsql.Wrap(&sqlite3.SQLiteDriver{}, otelsql.WithAllWrapperOptions()))
	}

	// Connect to a SQLite3 database using the otelsql driver wrapper.
	db, err := sql.Open("sqlite3-otel", "resource.db")
	if err != nil {
		log.Fatal(err)
	}

	age := 27
	rows, err := db.QueryContext(context.Background(), "SELECT name FROM users WHERE age=?", age)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	names := make([]string, 0)

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			// Check for a scan error.
			// Query rows will be closed with defer.
			log.Fatal(err)
		}
		names = append(names, name)
	}
	// If the database is being written to ensure to check for Close
	// errors that may be returned from the driver. The query may
	// encounter an auto-commit error and be forced to rollback changes.
	rerr := rows.Close()
	if rerr != nil {
		log.Fatal(rerr)
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s are %d years old", strings.Join(names, ", "), age)
}

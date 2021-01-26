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

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"

	"go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql"
)

const instrumentationName = "go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql/example"

var mysqlDSN = "root:otel_password@tcp(mysql)/db?parseTime=true"

func initTracer() {
	exporter, err := stdout.NewExporter(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	cfg := sdktrace.Config{
		DefaultSampler: sdktrace.AlwaysSample(),
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithConfig(cfg),
		sdktrace.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)
}

func main() {
	initTracer()

	// Register an OTel driver
	driverName, err := otelsql.Register("mysql", semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		panic(err)
	}

	// Connect to database
	db, err := sql.Open(driverName, mysqlDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = query(db)
	if err != nil {
		panic(err)
	}
}

func query(db *sql.DB) error {
	// Create a span
	tracer := otel.GetTracerProvider()
	ctx, span := tracer.Tracer(instrumentationName).Start(context.Background(), "example")
	defer span.End()

	// Make a query
	rows, err := db.QueryContext(ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var currentTime time.Time
	for rows.Next() {
		err = rows.Scan(&currentTime)
		if err != nil {
			return err
		}
	}
	fmt.Println(currentTime)
	return nil
}

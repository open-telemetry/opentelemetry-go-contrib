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
	"fmt"
	"log"

	oteltrace "go.opentelemetry.io/otel/trace"

	oteltracestdout "go.opentelemetry.io/otel/exporters/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go.opentelemetry.io/contrib/instrumentation/github.com/go-gorm/gorm/otelgorm"
)

type Product struct {
	gorm.Model
	Code  string
	Price uint
}

const dbName = "test.db"

func initTracer() oteltrace.TracerProvider {
	exporter, err := oteltracestdout.NewExporter(oteltracestdout.WithPrettyPrint())
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
	if err != nil {
		log.Fatal(err)
	}

	return tp
}

func doGormOperations(ctx context.Context, db *gorm.DB) {
	p := &Product{Code: "D42", Price: 100}

	db = db.WithContext(ctx)

	// Create
	if tx := db.Create(p); tx.Error != nil {
		panic(tx.Error.Error())
	}

	// Update
	p.Price = 200
	if tx := db.Updates(p); tx.Error != nil {
		panic(tx.Error.Error())
	}

	// Read
	var product Product
	if tx := db.First(&product, 1); tx.Error != nil {
		panic(tx.Error.Error())
	}

	if tx := db.First(&product, "code = ?", "D42"); tx.Error != nil {
		panic(tx.Error.Error())
	}

	// Delete
	if tx := db.Delete(p); tx.Error != nil {
		panic(tx.Error.Error())
	}

	// this select should fail due to invalid table
	db.Exec("SELECT * FROM not_found")
}

func main() {
	tp := initTracer()
	ctx := context.Background()

	// Initialize db connection
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Initialize otel plugin with options
	plugin := otelgorm.NewPlugin(
		otelgorm.WithTracerProvider(tp),
		otelgorm.WithDBName(dbName))

	err = db.Use(plugin)
	if err != nil {
		panic("failed configuring plugin")
	}

	// Migrate the schema
	err = db.AutoMigrate(&Product{})
	if err != nil {
		panic(err.Error())
	}

	doGormOperations(ctx, db)

	fmt.Println("Done")
}

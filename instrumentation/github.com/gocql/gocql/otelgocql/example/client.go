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

// This example will create the keyspace
// "gocql_integration_example" and a single table
// with the following schema:
// gocql_integration_example.book
//   id UUID
//   title text
//   author_first_name text
//   author_last_name text
//   PRIMARY KEY(id)
// The example will insert fictional books into the database and
// then truncate the table.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gocql/gocql"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	zipkintrace "go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/metric/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/trace"
)

const keyspace = "gocql_integration_example"

var wg sync.WaitGroup

func main() {
	initMetrics()
	tp := initTracer()
	defer func() { tp.Shutdown(context.Background()) }() //nolint:errcheck
	initDb()

	ctx, span := otel.Tracer(
		"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/example",
	).Start(context.Background(), "begin example")

	cluster := getCluster()
	// Create a session to begin making queries
	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
	)
	if err != nil {
		log.Fatalf("failed to create a session, %v", err)
	}
	defer session.Close()

	batch := session.NewBatch(gocql.LoggedBatch)
	for i := 0; i < 500; i++ {
		batch.Query(
			"INSERT INTO book (id, title, author_first_name, author_last_name) VALUES (?, ?, ?, ?)",
			gocql.TimeUUID(),
			fmt.Sprintf("Example Book %d", i),
			"firstName",
			"lastName",
		)
	}
	if err := session.ExecuteBatch(batch.WithContext(ctx)); err != nil {
		log.Printf("failed to batch insert, %v", err)
	}

	res := session.Query(
		"SELECT title, author_first_name, author_last_name from book WHERE author_last_name = ?",
		"lastName",
	).WithContext(ctx).PageSize(100).Iter()

	var (
		title     string
		firstName string
		lastName  string
	)

	for res.Scan(&title, &firstName, &lastName) {
		res.Scan(&title, &firstName, &lastName)
	}

	res.Close()

	if err = session.Query("truncate table book").WithContext(ctx).Exec(); err != nil {
		log.Printf("failed to delete data, %v", err)
	}

	span.End()

	wg.Wait()
}

func initMetrics() {
	// Start prometheus
	cont := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries([]float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10}),
			),
			export.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
	)
	metricExporter, err := prometheus.New(prometheus.Config{}, cont)
	if err != nil {
		log.Fatalf("failed to install metric exporter, %v", err)
	}
	global.SetMeterProvider(metricExporter.MeterProvider())

	server := http.Server{Addr: ":2222"}
	http.HandleFunc("/", metricExporter.ServeHTTP)
	go func() {
		defer wg.Done()
		wg.Add(1)
		log.Print(server.ListenAndServe())
	}()

	// ctrl+c will stop the server gracefully
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	go func() {
		<-shutdown
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("problem shutting down server, %v", err)
		} else {
			log.Print("gracefully shutting down server")
		}
		err := cont.Stop(context.Background())
		if err != nil {
			log.Printf("error stopping metric controller: %s", err)
		}
	}()
}

func initTracer() *trace.TracerProvider {
	exporter, err := zipkintrace.New("http://localhost:9411/api/v2/spans")
	if err != nil {
		log.Fatalf("failed to create zipkin exporter: %s", err)
	}

	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)

	return tp
}

func initDb() {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "system"
	cluster.Consistency = gocql.LocalQuorum
	cluster.Timeout = time.Second * 2
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	stmt := fmt.Sprintf(
		"create keyspace if not exists %s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }",
		keyspace,
	)
	if err := session.Query(stmt).Exec(); err != nil {
		log.Fatal(err)
	}

	cluster.Keyspace = keyspace
	session, err = cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}

	stmt = "create table if not exists book(id UUID, title text, author_first_name text, author_last_name text, PRIMARY KEY(id))"
	if err = session.Query(stmt).Exec(); err != nil {
		log.Fatal(err)
	}

	if err := session.Query("create index if not exists on book(author_last_name)").Exec(); err != nil {
		log.Fatal(err)
	}
}

func getCluster() *gocql.ClusterConfig {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.LocalQuorum
	cluster.ProtoVersion = 3
	cluster.Timeout = 2 * time.Second
	return cluster
}

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

//go:build go1.18
// +build go1.18

package main

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
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql" //nolint:staticcheck // This is deprecated and will be removed in the next release.
)

const keyspace = "gocql_integration_example"

var wg sync.WaitGroup

func main() {
	if err := initMetrics(); err != nil {
		log.Fatalf("failed to install metric exporter, %v", err)
	}
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to create zipkin exporter: %s", err)
	}
	defer func() { tp.Shutdown(context.Background()) }() //nolint:revive,errcheck
	if err := initDb(); err != nil {
		log.Fatal(err)
	}

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

func views() []metric.View {
	return []metric.View{
		metric.NewView(
			metric.Instrument{
				Name: "db.cassandra.rows",
			},
			metric.Stream{
				Aggregation: aggregation.ExplicitBucketHistogram{
					Boundaries: []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10},
				},
			},
		),
		metric.NewView(
			metric.Instrument{Name: "db.cassandra.latency"},
			metric.Stream{
				Aggregation: aggregation.ExplicitBucketHistogram{
					Boundaries: []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10},
				},
			},
		),
	}
}

func initMetrics() error {
	vs := views()

	exporter, err := otelprom.New()
	if err != nil {
		return err
	}
	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithView(vs...),
	)
	otel.SetMeterProvider(provider)

	http.Handle("/", promhttp.Handler())
	log.Print("Serving metrics at :2222/")
	go func() {
		err := http.ListenAndServe(":2222", nil)
		if err != nil {
			log.Print(err)
		}
	}()

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		err := provider.Shutdown(context.Background())
		if err != nil {
			log.Printf("error stopping MeterProvider: %s", err)
		}
	}()
	return nil
}

func initTracer() (*trace.TracerProvider, error) {
	exporter, err := zipkin.New("http://localhost:9411/api/v2/spans")
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)

	return tp, nil
}

func initDb() error {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "system"
	cluster.Consistency = gocql.LocalQuorum
	cluster.Timeout = time.Second * 2
	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}
	stmt := fmt.Sprintf(
		"create keyspace if not exists %s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }",
		keyspace,
	)
	if err := session.Query(stmt).Exec(); err != nil {
		return err
	}

	cluster.Keyspace = keyspace
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}

	stmt = "create table if not exists book(id UUID, title text, author_first_name text, author_last_name text, PRIMARY KEY(id))"
	if err = session.Query(stmt).Exec(); err != nil {
		return err
	}

	return session.Query("create index if not exists on book(author_last_name)").Exec()
}

func getCluster() *gocql.ClusterConfig {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.LocalQuorum
	cluster.ProtoVersion = 3
	cluster.Timeout = 2 * time.Second
	return cluster
}

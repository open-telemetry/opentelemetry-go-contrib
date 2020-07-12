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

// The Cassandra docker container conatains the
// "gocql-integration-example" keyspace and a single table
// with the following schema:
// gocql_integration_example.book
//   id UUID
//   title text
//   author_first_name text
//   author_last_name text
//   PRIMARY KEY(id)
// The example will insert fictional books into the database.

package main

import (
	"context"
	"log"

	"github.com/gocql/gocql"

	otelGocql "go.opentelemetry.io/contrib/github.com/gocql/gocql"
	"go.opentelemetry.io/otel/api/global"
	traceStdout "go.opentelemetry.io/otel/exporters/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
	traceExporter, err := traceStdout.NewExporter(traceStdout.Options{
		PrettyPrint: true,
	})
	if err != nil {
		log.Fatalf("failed to create span exporter, %v", err)
	}

	provider, err := sdktrace.NewProvider(
		sdktrace.WithSyncer(traceExporter),
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	if err != nil {
		log.Fatalf("failed to create trace provider, %v", err)
	}

	global.SetTraceProvider(provider)
}

func getCluster() *gocql.ClusterConfig {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "gocql_integration_example"
	cluster.Consistency = gocql.LocalQuorum
	cluster.ProtoVersion = 3
	return cluster
}

func main() {
	initTracer()

	ctx, span := global.Tracer(
		"go.opentelemetry.io/contrib/github.com/gocql/gocql/example",
	).Start(context.Background(), "begin example")

	cluster := getCluster()
	// Create a session to begin making queries
	session, err := otelGocql.NewSessionWithTracing(cluster)
	if err != nil {
		log.Fatalf("failed to create a session, %v", err)
	}
	defer session.Close()

	id := gocql.TimeUUID()
	if err := session.Query(
		"INSERT INTO book (id, title, author_first_name, author_last_name) VALUES (?, ?, ?, ?)",
		id,
		"Example Book 1",
		"firstName",
		"lastName",
	).WithContext(ctx).Exec(); err != nil {
		log.Fatalf("failed to insert data, %v", err)
	}

	res := session.Query(
		"SELECT title, author_first_name, author_last_name from book WHERE id = ?",
		id,
	).WithContext(ctx).Iter()

	var (
		title     string
		firstName string
		lastName  string
	)

	res.Scan(&title, &firstName, &lastName)

	log.Printf("Found Book {id: %s, title: %s, Name: %s, %s}", id, title, lastName, firstName)

	res.Close()

	if err = session.Query("DELETE FROM book WHERE id = ?", id).WithContext(ctx).Exec(); err != nil {
		log.Fatalf("failed to delete data, %v", err)
	}

	span.End()
}

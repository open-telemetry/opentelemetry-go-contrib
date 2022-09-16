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

package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/internal"
	"go.opentelemetry.io/contrib/internal/util"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// TODO(#2761): Add metric integration tests for the instrumentation. These
// tests depend on
// https://github.com/open-telemetry/opentelemetry-go/issues/3031 being
// resolved.

const (
	keyspace  string = "gotest"
	tableName string = "test_table"
)

type mockConnectObserver struct {
	callCount int
}

func (m *mockConnectObserver) ObserveConnect(observedConnect gocql.ObservedConnect) {
	m.callCount++
}

func TestQuery(t *testing.T) {
	defer afterEach(t)
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	ctx, parentSpan := tracerProvider.Tracer(internal.InstrumentationName).Start(context.Background(), "gocql-test")

	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
		otelgocql.WithTracerProvider(tracerProvider),
		otelgocql.WithConnectInstrumentation(false),
	)
	require.NoError(t, err)
	defer session.Close()
	require.NoError(t, session.AwaitSchemaAgreement(ctx))

	id := gocql.TimeUUID()
	title := "example-title"
	insertStmt := fmt.Sprintf("insert into %s (id, title) values (?, ?)", tableName)
	query := session.Query(insertStmt, id, title).WithContext(ctx)
	assert.NotNil(t, query, "expected query to not be nil")
	require.NoError(t, query.Exec())

	parentSpan.End()

	// Get the spans and ensure that they are child spans to the local parent
	spans := sr.Ended()

	// Collect all the connection spans
	// total spans:
	// 1 span for the Query
	// 1 span created in test
	require.Len(t, spans, 2)

	// Verify attributes are correctly added to the spans. Omit the one local span
	for _, span := range spans[0 : len(spans)-1] {
		switch span.Name() {
		case insertStmt:
			assert.Contains(t, span.Attributes(), semconv.DBStatementKey.String(insertStmt))
			assert.Equal(t, parentSpan.SpanContext().SpanID().String(), span.Parent().SpanID().String())
		default:
			t.Fatalf("unexpected span name %s", span.Name())
		}
		assertConnectionLevelAttributes(t, span)
	}
}

func TestBatch(t *testing.T) {
	defer afterEach(t)
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	ctx, parentSpan := tracerProvider.Tracer(internal.InstrumentationName).Start(context.Background(), "gocql-test")

	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
		otelgocql.WithTracerProvider(tracerProvider),
		otelgocql.WithConnectInstrumentation(false),
	)
	require.NoError(t, err)
	defer session.Close()
	require.NoError(t, session.AwaitSchemaAgreement(ctx))

	batch := session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for i := 0; i < 10; i++ {
		id := gocql.TimeUUID()
		title := fmt.Sprintf("batch-title-%d", i)
		stmt := fmt.Sprintf("insert into %s (id, title) values (?, ?)", tableName)
		batch.Query(stmt, id, title)
	}

	require.NoError(t, session.ExecuteBatch(batch))

	parentSpan.End()

	spans := sr.Ended()
	// total spans:
	// 1 span for the query
	// 1 span for the local span
	if assert.Len(t, spans, 2) {
		span := spans[0]
		assert.Equal(t, internal.CassBatchQueryName, span.Name())
		assert.Equal(t, parentSpan.SpanContext().SpanID(), span.Parent().SpanID())
		assert.Contains(t, span.Attributes(), semconv.DBOperationKey.String("db.cassandra.batch.query"))
		assertConnectionLevelAttributes(t, span)
	}
}

func TestConnection(t *testing.T) {
	defer afterEach(t)
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	connectObserver := &mockConnectObserver{0}
	ctx := context.Background()

	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
		otelgocql.WithTracerProvider(tracerProvider),
		otelgocql.WithConnectObserver(connectObserver),
	)
	require.NoError(t, err)
	defer session.Close()
	require.NoError(t, session.AwaitSchemaAgreement(ctx))

	spans := sr.Ended()

	assert.Less(t, 0, connectObserver.callCount)

	// Verify the span attributes
	for _, span := range spans {
		assert.Equal(t, internal.CassConnectName, span.Name())
		assert.Contains(t, span.Attributes(), semconv.DBOperationKey.String("db.cassandra.connect"))
		assertConnectionLevelAttributes(t, span)
	}
}

func TestHostOrIP(t *testing.T) {
	hostAndPort := "127.0.0.1:9042"
	attr := internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerIPKey, attr.Key)
	assert.Equal(t, "127.0.0.1", attr.Value.AsString())

	hostAndPort = "exampleHost:9042"
	attr = internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerNameKey, attr.Key)
	assert.Equal(t, "exampleHost", attr.Value.AsString())

	hostAndPort = "invalid-host-and-port-string"
	attr = internal.HostOrIP(hostAndPort)
	require.Empty(t, attr.Value.AsString())
}

func assertConnectionLevelAttributes(t *testing.T, span sdktrace.ReadOnlySpan) {
	attrs := span.Attributes()
	assert.Contains(t, attrs, semconv.DBSystemCassandra)
	assert.Contains(t, attrs, semconv.NetPeerIPKey.String("127.0.0.1"))
	assert.Contains(t, attrs, semconv.NetPeerPortKey.Int64(9042))
	assert.Contains(t, attrs, internal.CassHostStateKey.String("UP"))
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())

	keys := make(map[attribute.Key]struct{}, len(attrs))
	for _, a := range attrs {
		keys[a.Key] = struct{}{}
	}
	assert.Contains(t, keys, internal.CassVersionKey)
	assert.Contains(t, keys, internal.CassHostIDKey)
}

// getCluster creates a gocql ClusterConfig with the appropriate
// settings for test cases.
func getCluster() *gocql.ClusterConfig {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.LocalQuorum
	cluster.NumConns = 1
	return cluster
}

// beforeAll creates the testing keyspace and table if they do not already exist.
func beforeAll() error {
	cluster := gocql.NewCluster("localhost")
	cluster.Consistency = gocql.LocalQuorum
	cluster.Keyspace = "system"

	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("failed to connect to database during beforeAll, %v", err)
	}

	err = session.Query(
		fmt.Sprintf(
			"create keyspace if not exists %s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }",
			keyspace,
		),
	).Exec()
	if err != nil {
		return err
	}

	cluster.Keyspace = keyspace
	cluster.Timeout = time.Second * 2
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}

	err = session.Query(
		fmt.Sprintf("create table if not exists %s(id UUID, title text, PRIMARY KEY(id))", tableName),
	).Exec()
	if err != nil {
		return err
	}
	return nil
}

// afterEach truncates the table used for testing.
func afterEach(t *testing.T) {
	cluster := gocql.NewCluster("localhost")
	cluster.Consistency = gocql.LocalQuorum
	cluster.Keyspace = keyspace
	cluster.Timeout = time.Second * 2
	session, err := cluster.CreateSession()
	if err != nil {
		t.Fatalf("failed to connect to database during afterEach, %v", err)
	}
	if err = session.Query(fmt.Sprintf("truncate table %s", tableName)).Exec(); err != nil {
		t.Fatalf("failed to truncate table, %v", err)
	}
}

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-gocql")
	if err := beforeAll(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

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
	"go.opentelemetry.io/otel/metric/metrictest"
	"go.opentelemetry.io/otel/metric/number"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

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

type testRecord struct {
	name       string
	meterName  string
	attributes []attribute.KeyValue
	number     number.Number
	numberKind number.Kind
}

func TestQuery(t *testing.T) {
	defer afterEach()
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	meterProvider := metrictest.NewMeterProvider()

	ctx, parentSpan := tracerProvider.Tracer(internal.InstrumentationName).Start(context.Background(), "gocql-test")

	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
		otelgocql.WithTracerProvider(tracerProvider),
		otelgocql.WithMeterProvider(meterProvider),
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

	// Check metrics
	actual := obtainTestRecords(meterProvider.MeasurementBatches)
	require.Len(t, actual, 3)
	expected := []testRecord{
		{
			name:      "db.cassandra.queries",
			meterName: internal.InstrumentationName,
			attributes: []attribute.KeyValue{
				internal.CassDBSystem(),
				internal.CassPeerIP("127.0.0.1"),
				internal.CassPeerPort(9042),
				internal.CassVersion("3"),
				internal.CassHostID("test-id"),
				internal.CassHostState("UP"),
				internal.CassKeyspace(keyspace),
				internal.CassStatement(insertStmt),
			},
			number: 1,
		},
		{
			name:      "db.cassandra.rows",
			meterName: internal.InstrumentationName,
			attributes: []attribute.KeyValue{
				internal.CassDBSystem(),
				internal.CassPeerIP("127.0.0.1"),
				internal.CassPeerPort(9042),
				internal.CassVersion("3"),
				internal.CassHostID("test-id"),
				internal.CassHostState("UP"),
				internal.CassKeyspace(keyspace),
			},
			number: 0,
		},
		{
			name:      "db.cassandra.latency",
			meterName: internal.InstrumentationName,
			attributes: []attribute.KeyValue{
				internal.CassDBSystem(),
				internal.CassPeerIP("127.0.0.1"),
				internal.CassPeerPort(9042),
				internal.CassVersion("3"),
				internal.CassHostID("test-id"),
				internal.CassHostState("UP"),
				internal.CassKeyspace(keyspace),
			},
		},
	}

	for _, record := range actual {
		switch record.name {
		case "db.cassandra.queries":
			recordEqual(t, expected[0], record)
			assert.Equal(t, expected[0].number, record.number)
		case "db.cassandra.rows":
			recordEqual(t, expected[1], record)
			assert.Equal(t, expected[1].number, record.number)
		case "db.cassandra.latency":
			recordEqual(t, expected[2], record)
			// The latency will vary, so just check that it exists
			assert.True(t, !record.number.IsZero(record.numberKind))
		default:
			t.Fatalf("wrong metric %s", record.name)
		}
	}

}

func TestBatch(t *testing.T) {
	defer afterEach()
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	meterProvider := metrictest.NewMeterProvider()

	ctx, parentSpan := tracerProvider.Tracer(internal.InstrumentationName).Start(context.Background(), "gocql-test")

	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
		otelgocql.WithTracerProvider(tracerProvider),
		otelgocql.WithMeterProvider(meterProvider),
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

	// Check metrics
	actual := obtainTestRecords(meterProvider.MeasurementBatches)
	require.Len(t, actual, 2)
	expected := []testRecord{
		{
			name:      "db.cassandra.batch.queries",
			meterName: internal.InstrumentationName,
			attributes: []attribute.KeyValue{
				internal.CassDBSystem(),
				internal.CassPeerIP("127.0.0.1"),
				internal.CassPeerPort(9042),
				internal.CassVersion("3"),
				internal.CassHostID("test-id"),
				internal.CassHostState("UP"),
				internal.CassKeyspace(keyspace),
			},
			number: 1,
		},
		{
			name:      "db.cassandra.latency",
			meterName: internal.InstrumentationName,
			attributes: []attribute.KeyValue{
				internal.CassDBSystem(),
				internal.CassPeerIP("127.0.0.1"),
				internal.CassPeerPort(9042),
				internal.CassVersion("3"),
				internal.CassHostID("test-id"),
				internal.CassHostState("UP"),
				internal.CassKeyspace(keyspace),
			},
		},
	}

	for _, record := range actual {
		switch record.name {
		case "db.cassandra.batch.queries":
			recordEqual(t, expected[0], record)
			assert.Equal(t, expected[0].number, record.number)
		case "db.cassandra.latency":
			recordEqual(t, expected[1], record)
			assert.True(t, !record.number.IsZero(record.numberKind))
		default:
			t.Fatalf("wrong metric %s", record.name)
		}
	}

}

func TestConnection(t *testing.T) {
	defer afterEach()
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	meterProvider := metrictest.NewMeterProvider()
	connectObserver := &mockConnectObserver{0}
	ctx := context.Background()

	session, err := otelgocql.NewSessionWithTracing(
		ctx,
		cluster,
		otelgocql.WithTracerProvider(tracerProvider),
		otelgocql.WithMeterProvider(meterProvider),
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

	// Verify the metrics
	actual := obtainTestRecords(meterProvider.MeasurementBatches)
	expected := []testRecord{
		{
			name:      "db.cassandra.connections",
			meterName: internal.InstrumentationName,
			attributes: []attribute.KeyValue{
				internal.CassDBSystem(),
				internal.CassPeerIP("127.0.0.1"),
				internal.CassPeerPort(9042),
				internal.CassVersion("3"),
				internal.CassHostID("test-id"),
				internal.CassHostState("UP"),
			},
		},
	}

	for _, record := range actual {
		switch record.name {
		case "db.cassandra.connections":
			recordEqual(t, expected[0], record)
		default:
			t.Fatalf("wrong metric %s", record.name)
		}
	}
}

func TestHostOrIP(t *testing.T) {
	hostAndPort := "127.0.0.1:9042"
	attribute := internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerIPKey, attribute.Key)
	assert.Equal(t, "127.0.0.1", attribute.Value.AsString())

	hostAndPort = "exampleHost:9042"
	attribute = internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerNameKey, attribute.Key)
	assert.Equal(t, "exampleHost", attribute.Value.AsString())

	hostAndPort = "invalid-host-and-port-string"
	attribute = internal.HostOrIP(hostAndPort)
	require.Empty(t, attribute.Value.AsString())
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

// obtainTestRecords creates a slice of testRecord with values
// obtained from measurements
func obtainTestRecords(mbs []metrictest.Batch) []testRecord {
	var records []testRecord
	for _, mb := range mbs {
		for _, m := range mb.Measurements {
			records = append(
				records,
				testRecord{
					name:       m.Instrument.Descriptor().Name(),
					meterName:  mb.Library.InstrumentationName,
					attributes: mb.Labels,
					number:     m.Number,
					numberKind: m.Instrument.Descriptor().NumberKind(),
				},
			)
		}
	}

	return records
}

// recordEqual checks that the given metric name and instrumentation names are equal.
func recordEqual(t *testing.T, expected testRecord, actual testRecord) {
	assert.Equal(t, expected.name, actual.name)
	assert.Equal(t, expected.meterName, actual.meterName)
	require.Len(t, actual.attributes, len(expected.attributes))
	actualSet := attribute.NewSet(actual.attributes...)
	for _, attribute := range expected.attributes {
		actualValue, ok := actualSet.Value(attribute.Key)
		assert.True(t, ok)
		assert.NotNil(t, actualValue)
		// Can't test equality of host id
		if attribute.Key != internal.CassHostIDKey && attribute.Key != internal.CassVersionKey {
			assert.Equal(t, attribute.Value, actualValue)
		} else {
			assert.NotEmpty(t, actualValue)
		}
	}
}

// beforeAll creates the testing keyspace and table if they do not already exist.
func beforeAll() {
	cluster := gocql.NewCluster("localhost")
	cluster.Consistency = gocql.LocalQuorum
	cluster.Keyspace = "system"

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("failed to connect to database during beforeAll, %v", err)
	}

	err = session.Query(
		fmt.Sprintf(
			"create keyspace if not exists %s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }",
			keyspace,
		),
	).Exec()
	if err != nil {
		log.Fatal(err)
	}

	cluster.Keyspace = keyspace
	cluster.Timeout = time.Second * 2
	session, err = cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}

	err = session.Query(
		fmt.Sprintf("create table if not exists %s(id UUID, title text, PRIMARY KEY(id))", tableName),
	).Exec()
	if err != nil {
		log.Fatal(err)
	}
}

// afterEach truncates the table used for testing.
func afterEach() {
	cluster := gocql.NewCluster("localhost")
	cluster.Consistency = gocql.LocalQuorum
	cluster.Keyspace = keyspace
	cluster.Timeout = time.Second * 2
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("failed to connect to database during afterEach, %v", err)
	}
	if err = session.Query(fmt.Sprintf("truncate table %s", tableName)).Exec(); err != nil {
		log.Fatalf("failed to truncate table, %v", err)
	}
}

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-gocql")
	beforeAll()
	os.Exit(m.Run())
}

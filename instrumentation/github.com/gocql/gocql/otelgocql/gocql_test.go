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

package otelgocql

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/semconv"
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
	sr := new(oteltest.SpanRecorder)
	tracerProvider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	meterImpl, meterProvider := oteltest.NewMeterProvider()

	ctx, parentSpan := tracerProvider.Tracer(instrumentationName).Start(context.Background(), "gocql-test")

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
		WithConnectInstrumentation(false),
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
	spans := sr.Completed()

	// Collect all the connection spans
	// total spans:
	// 1 span for the Query
	// 1 span created in test
	require.Len(t, spans, 2)

	// Verify attributes are correctly added to the spans. Omit the one local span
	for _, span := range spans[0 : len(spans)-1] {

		switch span.Name() {
		case insertStmt:
			assert.Equal(t, insertStmt, span.Attributes()[semconv.DBStatementKey].AsString())
			assert.Equal(t, parentSpan.SpanContext().SpanID().String(), span.ParentSpanID().String())
		default:
			t.Fatalf("unexpected span name %s", span.Name())
		}
		assertConnectionLevelAttributes(t, span)
	}

	// Check metrics
	actual := obtainTestRecords(meterImpl.MeasurementBatches)
	require.Len(t, actual, 3)
	expected := []testRecord{
		{
			name:      "db.cassandra.queries",
			meterName: instrumentationName,
			attributes: []attribute.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
				cassStatement(insertStmt),
			},
			number: 1,
		},
		{
			name:      "db.cassandra.rows",
			meterName: instrumentationName,
			attributes: []attribute.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
			},
			number: 0,
		},
		{
			name:      "db.cassandra.latency",
			meterName: instrumentationName,
			attributes: []attribute.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
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
	sr := new(oteltest.SpanRecorder)
	tracerProvider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	meterImpl, meterProvider := oteltest.NewMeterProvider()

	ctx, parentSpan := tracerProvider.Tracer(instrumentationName).Start(context.Background(), "gocql-test")

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
		WithConnectInstrumentation(false),
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

	spans := sr.Completed()
	// total spans:
	// 1 span for the query
	// 1 span for the local span
	if assert.Len(t, spans, 2) {
		span := spans[0]
		assert.Equal(t, cassBatchQueryName, span.Name())
		assert.Equal(t, parentSpan.SpanContext().SpanID, span.ParentSpanID())
		assert.Equal(t, "db.cassandra.batch.query",
			span.Attributes()[semconv.DBOperationKey].AsString(),
		)
		assertConnectionLevelAttributes(t, span)
	}

	// Check metrics
	actual := obtainTestRecords(meterImpl.MeasurementBatches)
	require.Len(t, actual, 2)
	expected := []testRecord{
		{
			name:      "db.cassandra.batch.queries",
			meterName: instrumentationName,
			attributes: []attribute.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
			},
			number: 1,
		},
		{
			name:      "db.cassandra.latency",
			meterName: instrumentationName,
			attributes: []attribute.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
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
	sr := new(oteltest.SpanRecorder)
	tracerProvider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	meterImpl, meterProvider := oteltest.NewMeterProvider()
	connectObserver := &mockConnectObserver{0}
	ctx := context.Background()

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
		WithConnectObserver(connectObserver),
	)
	require.NoError(t, err)
	defer session.Close()
	require.NoError(t, session.AwaitSchemaAgreement(ctx))

	spans := sr.Completed()

	assert.Less(t, 0, connectObserver.callCount)

	// Verify the span attributes
	for _, span := range spans {
		assert.Equal(t, cassConnectName, span.Name())
		assert.Equal(t, "db.cassandra.connect", span.Attributes()[semconv.DBOperationKey].AsString())
		assertConnectionLevelAttributes(t, span)
	}

	// Verify the metrics
	actual := obtainTestRecords(meterImpl.MeasurementBatches)
	expected := []testRecord{
		{
			name:      "db.cassandra.connections",
			meterName: instrumentationName,
			attributes: []attribute.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
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
	attribute := hostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerIPKey, attribute.Key)
	assert.Equal(t, "127.0.0.1", attribute.Value.AsString())

	hostAndPort = "exampleHost:9042"
	attribute = hostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerNameKey, attribute.Key)
	assert.Equal(t, "exampleHost", attribute.Value.AsString())

	hostAndPort = "invalid-host-and-port-string"
	attribute = hostOrIP(hostAndPort)
	require.Empty(t, attribute.Value.AsString())
}

func assertConnectionLevelAttributes(t *testing.T, span *oteltest.Span) {
	assert.Equal(t, span.Attributes()[semconv.DBSystemKey].AsString(),
		semconv.DBSystemCassandra.Value.AsString(),
	)
	assert.Equal(t, "127.0.0.1", span.Attributes()[semconv.NetPeerIPKey].AsString())
	assert.Equal(t, int64(9042), span.Attributes()[semconv.NetPeerPortKey].AsInt64())
	assert.Contains(t, span.Attributes(), cassVersionKey)
	assert.Contains(t, span.Attributes(), cassHostIDKey)
	assert.Equal(t, "up", strings.ToLower(span.Attributes()[cassHostStateKey].AsString()))
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())
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
func obtainTestRecords(mbs []oteltest.Batch) []testRecord {
	var records []testRecord
	for _, mb := range mbs {
		for _, m := range mb.Measurements {
			records = append(
				records,
				testRecord{
					name:       m.Instrument.Descriptor().Name(),
					meterName:  m.Instrument.Descriptor().InstrumentationName(),
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
		if attribute.Key != cassHostIDKey && attribute.Key != cassVersionKey {
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

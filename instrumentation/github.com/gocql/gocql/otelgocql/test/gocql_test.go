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
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
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

func TestQuery(t *testing.T) {
	defer afterEach(t)
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))

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
			assert.Contains(t, span.Attributes(), semconv.DBStatement(insertStmt))
			assert.Equal(t, parentSpan.SpanContext().SpanID().String(), span.Parent().SpanID().String())
		default:
			t.Fatalf("unexpected span name %s", span.Name())
		}
		assertConnectionLevelAttributes(t, span)
	}

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]
	assertScope(t, sm)
	assertQueriesMetric(t, 1, insertStmt, requireMetric(t, "db.cassandra.queries", sm.Metrics))
	assertRowsMetric(t, 1, requireMetric(t, "db.cassandra.rows", sm.Metrics))
	assertLatencyMetric(t, 1, requireMetric(t, "db.cassandra.latency", sm.Metrics))
}

func TestBatch(t *testing.T) {
	defer afterEach(t)
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))

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
		assert.Contains(t, span.Attributes(), semconv.DBOperation("db.cassandra.batch.query"))
		assertConnectionLevelAttributes(t, span)
	}

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]
	assertScope(t, sm)
	assertBatchQueriesMetric(t, 1, requireMetric(t, "db.cassandra.batch.queries", sm.Metrics))
	assertLatencyMetric(t, 1, requireMetric(t, "db.cassandra.latency", sm.Metrics))
}

func TestConnection(t *testing.T) {
	defer afterEach(t)
	cluster := getCluster()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
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
		assert.Contains(t, span.Attributes(), semconv.DBOperation("db.cassandra.connect"))
		assertConnectionLevelAttributes(t, span)
	}

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]
	assertScope(t, sm)
	assertConnectionsMetric(t, requireMetric(t, "db.cassandra.connections", sm.Metrics))
}

func TestHostOrIP(t *testing.T) {
	hostAndPort := "127.0.0.1:9042"
	attr := internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetSockPeerAddrKey, attr.Key)
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
	assert.Contains(t, attrs, semconv.NetSockPeerAddr("127.0.0.1"))
	assert.Contains(t, attrs, semconv.NetPeerPort(9042))
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

func assertScope(t *testing.T, sm metricdata.ScopeMetrics) {
	assert.Equal(t, instrumentation.Scope{
		Name:    "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql",
		Version: otelgocql.Version(),
	}, sm.Scope)
}

func requireMetric(t *testing.T, name string, metrics []metricdata.Metrics) metricdata.Metrics {
	m, ok := getMetric(name, metrics)
	require.Truef(t, ok, "missing metric %q", name)
	return m
}

func getMetric(name string, metrics []metricdata.Metrics) (metricdata.Metrics, bool) {
	for _, m := range metrics {
		if m.Name == name {
			return m, true
		}
	}
	return metricdata.Metrics{}, false
}

func assertQueriesMetric(t *testing.T, value int64, stmt string, m metricdata.Metrics) {
	assert.Equal(t, "db.cassandra.queries", m.Name)
	assert.Equal(t, "Number queries executed", m.Description)
	require.IsType(t, m.Data, metricdata.Sum[int64]{})
	data := m.Data.(metricdata.Sum[int64])
	assert.Equal(t, metricdata.CumulativeTemporality, data.Temporality, "Temporality")
	assert.True(t, data.IsMonotonic, "IsMonotonic")
	require.Len(t, data.DataPoints, 1, "DataPoints")
	dPt := data.DataPoints[0]
	assert.Equal(t, value, dPt.Value, "Value")
	assertAttrSet(t, []attribute.KeyValue{
		internal.CassDBSystem(),
		internal.CassPeerIP("127.0.0.1"),
		internal.CassPeerPort(9042),
		internal.CassVersion("3"),
		internal.CassHostID("test-id"),
		internal.CassHostState("UP"),
		internal.CassKeyspace(keyspace),
		internal.CassStatement(stmt),
	}, dPt.Attributes)
}

func assertBatchQueriesMetric(t *testing.T, value int64, m metricdata.Metrics) {
	assert.Equal(t, "db.cassandra.batch.queries", m.Name)
	assert.Equal(t, "Number of batch queries executed", m.Description)
	require.IsType(t, m.Data, metricdata.Sum[int64]{})
	data := m.Data.(metricdata.Sum[int64])
	assert.Equal(t, metricdata.CumulativeTemporality, data.Temporality, "Temporality")
	assert.True(t, data.IsMonotonic, "IsMonotonic")
	require.Len(t, data.DataPoints, 1, "DataPoints")
	dPt := data.DataPoints[0]
	assert.Equal(t, value, dPt.Value, "Value")
	assertAttrSet(t, []attribute.KeyValue{
		internal.CassDBSystem(),
		internal.CassPeerIP("127.0.0.1"),
		internal.CassPeerPort(9042),
		internal.CassVersion("3"),
		internal.CassHostID("test-id"),
		internal.CassHostState("UP"),
		internal.CassKeyspace(keyspace),
	}, dPt.Attributes)
}

func assertConnectionsMetric(t *testing.T, m metricdata.Metrics) {
	assert.Equal(t, "db.cassandra.connections", m.Name)
	assert.Equal(t, "Number of connections created", m.Description)
	require.IsType(t, m.Data, metricdata.Sum[int64]{})
	data := m.Data.(metricdata.Sum[int64])
	assert.Equal(t, metricdata.CumulativeTemporality, data.Temporality, "Temporality")
	assert.True(t, data.IsMonotonic, "IsMonotonic")
	for _, dPt := range data.DataPoints {
		assertAttrSet(t, []attribute.KeyValue{
			internal.CassDBSystem(),
			internal.CassPeerIP("127.0.0.1"),
			internal.CassPeerPort(9042),
			internal.CassVersion("3"),
			internal.CassHostID("test-id"),
			internal.CassHostState("UP"),
		}, dPt.Attributes)
	}
}

func assertRowsMetric(t *testing.T, count uint64, m metricdata.Metrics) {
	assert.Equal(t, "db.cassandra.rows", m.Name)
	assert.Equal(t, "Number of rows returned from query", m.Description)
	require.IsType(t, m.Data, metricdata.Histogram[int64]{})
	data := m.Data.(metricdata.Histogram[int64])
	assert.Equal(t, metricdata.CumulativeTemporality, data.Temporality, "Temporality")
	require.Len(t, data.DataPoints, 1, "DataPoints")
	dPt := data.DataPoints[0]
	assert.Equal(t, count, dPt.Count, "Count")
	assertAttrSet(t, []attribute.KeyValue{
		internal.CassDBSystem(),
		internal.CassPeerIP("127.0.0.1"),
		internal.CassPeerPort(9042),
		internal.CassVersion("3"),
		internal.CassHostID("test-id"),
		internal.CassHostState("UP"),
		internal.CassKeyspace(keyspace),
	}, dPt.Attributes)
}

func assertLatencyMetric(t *testing.T, count uint64, m metricdata.Metrics) {
	assert.Equal(t, "db.cassandra.latency", m.Name)
	assert.Equal(t, "Sum of latency to host in milliseconds", m.Description)
	assert.Equal(t, "ms", m.Unit)
	require.IsType(t, m.Data, metricdata.Histogram[int64]{})
	data := m.Data.(metricdata.Histogram[int64])
	assert.Equal(t, metricdata.CumulativeTemporality, data.Temporality, "Temporality")
	require.Len(t, data.DataPoints, 1, "DataPoints")
	dPt := data.DataPoints[0]
	assert.Equal(t, count, dPt.Count, "Count")
	assertAttrSet(t, []attribute.KeyValue{
		internal.CassDBSystem(),
		internal.CassPeerIP("127.0.0.1"),
		internal.CassPeerPort(9042),
		internal.CassVersion("3"),
		internal.CassHostID("test-id"),
		internal.CassHostState("UP"),
		internal.CassKeyspace(keyspace),
	}, dPt.Attributes)
}

func assertAttrSet(t *testing.T, want []attribute.KeyValue, got attribute.Set) {
	for _, attr := range want {
		actual, ok := got.Value(attr.Key)
		if !assert.Truef(t, ok, "missing attribute %s", attr.Key) {
			continue
		}
		switch attr.Key {
		case internal.CassHostIDKey, internal.CassVersionKey:
			// Host ID and Version will change between test runs.
			assert.NotEmpty(t, actual)
		default:
			assert.Equal(t, attr.Value, actual)
		}
	}
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

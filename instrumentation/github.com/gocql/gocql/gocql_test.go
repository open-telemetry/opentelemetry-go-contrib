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

package gocql

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/semconv"

	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/contrib/internal/util"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
)

const (
	keyspace  string = "gotest"
	tableName string = "test_table"
)

var exporter *mockExporter

// mockExporter provides an exporter to access metrics
// used for testing puporses only.
type mockExporter struct {
	t       *testing.T
	records []export.Record
}

func (mockExporter) ExportKindFor(*metric.Descriptor, aggregation.Kind) export.ExportKind {
	return export.PassThroughExporter
}

func (e *mockExporter) Export(_ context.Context, set export.CheckpointSet) error {
	if err := set.ForEach(e, func(record export.Record) error {
		e.records = append(e.records, record)
		return nil
	}); err != nil {
		e.t.Fatal(err)
		return err
	}
	return nil
}

// mockExportPipeline returns a push controller with a mockExporter.
func mockExportPipeline(t *testing.T) *push.Controller {
	var records []export.Record
	exporter = &mockExporter{t, records}
	controller := push.New(
		basic.New(
			simple.NewWithExactDistribution(),
			exporter,
		),
		exporter,
		push.WithPeriod(1*time.Second),
	)
	controller.Start()
	return controller
}

type mockTraceProvider struct {
	tracer *mocktracer.Tracer
}

func (p *mockTraceProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return p.tracer
}

func newTraceProvider() *mockTraceProvider {
	return &mockTraceProvider{
		mocktracer.NewTracer(instrumentationName),
	}
}

type mockConnectObserver struct {
	callCount int
}

func (m *mockConnectObserver) ObserveConnect(observedConnect gocql.ObservedConnect) {
	m.callCount++
}

type testRecord struct {
	Name      string
	MeterName string
	Labels    []label.KeyValue
	Number    metric.Number
}

func TestQuery(t *testing.T) {
	controller := getController(t)
	defer afterEach()
	cluster := getCluster()
	traceProvider := newTraceProvider()

	ctx, parentSpan := traceProvider.tracer.Start(context.Background(), "gocql-test")

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTraceProvider(traceProvider),
		WithMeterProvider(controller.Provider()),
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
	spans := traceProvider.tracer.EndedSpans()

	// Collect all the connection spans
	// total spans:
	// 1 span for the Query
	// 1 span created in test
	require.Len(t, spans, 2)

	// Verify attributes are correctly added to the spans. Omit the one local span
	for _, span := range spans[0 : len(spans)-1] {

		switch span.Name {
		case insertStmt:
			assert.Equal(t, insertStmt, span.Attributes[semconv.DBStatementKey].AsString())
			assert.Equal(t, parentSpan.SpanContext().SpanID.String(), span.ParentSpanID.String())
		default:
			t.Fatalf("unexpected span name %s", span.Name)
		}
		assertConnectionLevelAttributes(t, span)
	}

	// Check metrics
	controller.Stop()

	require.Len(t, exporter.records, 3)
	expected := []testRecord{
		{
			Name:      "db.cassandra.queries",
			MeterName: instrumentationName,
			Labels: []label.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
				cassStatement(insertStmt),
			},
			Number: 1,
		},
		{
			Name:      "db.cassandra.rows",
			MeterName: instrumentationName,
			Labels: []label.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
			},
			Number: 0,
		},
		{
			Name:      "db.cassandra.latency",
			MeterName: instrumentationName,
			Labels: []label.KeyValue{
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

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		agg := record.Aggregation()
		switch name {
		case "db.cassandra.queries":
			recordEqual(t, expected[0], record)
			numberEqual(t, expected[0].Number, agg)
		case "db.cassandra.rows":
			recordEqual(t, expected[1], record)
			numberEqual(t, expected[1].Number, agg)
		case "db.cassandra.latency":
			recordEqual(t, expected[2], record)
			// The latency will vary, so just check that it exists
			if _, ok := agg.(aggregation.MinMaxSumCount); !ok {
				t.Fatal("missing aggregation in latency record")
			}
		default:
			t.Fatalf("wrong metric %s", name)
		}
	}

}

func TestBatch(t *testing.T) {
	controller := getController(t)
	defer afterEach()
	cluster := getCluster()
	traceProvider := newTraceProvider()

	ctx, parentSpan := traceProvider.tracer.Start(context.Background(), "gocql-test")

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTraceProvider(traceProvider),
		WithMeterProvider(controller.Provider()),
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

	spans := traceProvider.tracer.EndedSpans()
	// total spans:
	// 1 span for the query
	// 1 span for the local span
	if assert.Len(t, spans, 2) {
		span := spans[0]
		assert.Equal(t, cassBatchQueryName, span.Name)
		assert.Equal(t, parentSpan.SpanContext().SpanID, span.ParentSpanID)
		assert.Equal(t, "db.cassandra.batch.query",
			span.Attributes[semconv.DBOperationKey].AsString(),
		)
		assertConnectionLevelAttributes(t, span)
	}

	controller.Stop()

	// Check metrics
	require.Len(t, exporter.records, 2)
	expected := []testRecord{
		{
			Name:      "db.cassandra.batch.queries",
			MeterName: instrumentationName,
			Labels: []label.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
				cassKeyspace(keyspace),
			},
			Number: 1,
		},
		{
			Name:      "db.cassandra.latency",
			MeterName: instrumentationName,
			Labels: []label.KeyValue{
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

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		agg := record.Aggregation()
		switch name {
		case "db.cassandra.batch.queries":
			recordEqual(t, expected[0], record)
			numberEqual(t, expected[0].Number, agg)
		case "db.cassandra.latency":
			recordEqual(t, expected[1], record)
			if _, ok := agg.(aggregation.MinMaxSumCount); !ok {
				t.Fatal("missing aggregation in latency record")
			}
		default:
			t.Fatalf("wrong metric %s", name)
		}
	}

}

func TestConnection(t *testing.T) {
	controller := getController(t)
	defer afterEach()
	cluster := getCluster()
	traceProvider := newTraceProvider()
	connectObserver := &mockConnectObserver{0}
	ctx := context.Background()

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTraceProvider(traceProvider),
		WithMeterProvider(controller.Provider()),
		WithConnectObserver(connectObserver),
	)
	require.NoError(t, err)
	defer session.Close()
	require.NoError(t, session.AwaitSchemaAgreement(ctx))

	spans := traceProvider.tracer.EndedSpans()

	assert.Less(t, 0, connectObserver.callCount)

	controller.Stop()

	// Verify the span attributes
	for _, span := range spans {
		assert.Equal(t, cassConnectName, span.Name)
		assert.Equal(t, "db.cassandra.connect", span.Attributes[semconv.DBOperationKey].AsString())
		assertConnectionLevelAttributes(t, span)
	}

	// Verify the metrics
	expected := []testRecord{
		{
			Name:      "db.cassandra.connections",
			MeterName: instrumentationName,
			Labels: []label.KeyValue{
				cassDBSystem(),
				cassPeerIP("127.0.0.1"),
				cassPeerPort(9042),
				cassVersion("3"),
				cassHostID("test-id"),
				cassHostState("UP"),
			},
		},
	}

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		switch name {
		case "db.cassandra.connections":
			recordEqual(t, expected[0], record)
		default:
			t.Fatalf("wrong metric %s", name)
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

func assertConnectionLevelAttributes(t *testing.T, span *mocktracer.Span) {
	assert.Equal(t, span.Attributes[semconv.DBSystemKey].AsString(),
		semconv.DBSystemCassandra.Value.AsString(),
	)
	assert.Equal(t, "127.0.0.1", span.Attributes[semconv.NetPeerIPKey].AsString())
	assert.Equal(t, int32(9042), span.Attributes[semconv.NetPeerPortKey].AsInt32())
	assert.Contains(t, span.Attributes, cassVersionKey)
	assert.Contains(t, span.Attributes, cassHostIDKey)
	assert.Equal(t, "up", strings.ToLower(span.Attributes[cassHostStateKey].AsString()))
	assert.Equal(t, trace.SpanKindClient, span.Kind)
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

// getController returns the push controller for the mock
// export pipeline.
func getController(t *testing.T) *push.Controller {
	controller := mockExportPipeline(t)
	return controller
}

// recordEqual checks that the given metric name and instrumentation names are equal.
func recordEqual(t *testing.T, expected testRecord, actual export.Record) {
	descriptor := actual.Descriptor()
	assert.Equal(t, expected.Name, descriptor.Name())
	assert.Equal(t, expected.MeterName, descriptor.InstrumentationName())
	require.Len(t, actual.Labels().ToSlice(), len(expected.Labels))
	for _, label := range expected.Labels {
		actualValue, ok := actual.Labels().Value(label.Key)
		assert.True(t, ok)
		assert.NotNil(t, actualValue)
		// Can't test equality of host id
		if label.Key != cassHostIDKey && label.Key != cassVersionKey {
			assert.Equal(t, label.Value, actualValue)
		} else {
			assert.NotEmpty(t, actualValue)
		}
	}
}

func numberEqual(t *testing.T, expected metric.Number, agg aggregation.Aggregation) {
	kind := agg.Kind()
	switch kind {
	case aggregation.SumKind:
		if sum, ok := agg.(aggregation.Sum); !ok {
			t.Fatal("missing sum value")
		} else {
			if num, err := sum.Sum(); err == nil {
				assert.Equal(t, expected, num)
			} else {
				t.Fatal("missing value")
			}
		}
	case aggregation.ExactKind:
		if mmsc, ok := agg.(aggregation.MinMaxSumCount); !ok {
			t.Fatal("missing aggregation")
		} else {
			if max, err := mmsc.Max(); err == nil {
				assert.Equal(t, expected, max)
			} else {
				t.Fatal("missing sum")
			}
		}
	default:
		t.Fatalf("unexpected kind %s", kind)
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

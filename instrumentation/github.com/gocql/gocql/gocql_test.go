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

	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
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
		simple.NewWithExactDistribution(),
		exporter,
		push.WithPeriod(1*time.Second),
	)
	controller.Start()
	return controller
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
	Labels    []kv.KeyValue
	Number    metric.Number
}

func TestQuery(t *testing.T) {
	controller := getController(t)
	defer afterEach()
	cluster := getCluster()
	tracer := mocktracer.NewTracer("gocql-test")

	ctx, parentSpan := tracer.Start(context.Background(), "gocql-test")

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTracer(tracer),
		WithConnectInstrumentation(false),
	)
	assert.NoError(t, err)
	defer session.Close()

	id := gocql.TimeUUID()
	title := "example-title"
	insertStmt := fmt.Sprintf("insert into %s (id, title) values (?, ?)", tableName)
	query := session.Query(insertStmt, id, title).WithContext(ctx)
	assert.NotNil(t, query, "expected query to not be nil")
	if err := query.Exec(); err != nil {
		t.Fatal(err.Error())
	}

	parentSpan.End()

	// Get the spans and ensure that they are child spans to the local parent
	spans := tracer.EndedSpans()

	// Collect all the connection spans
	// total spans:
	// 1 span for the Query
	// 1 span created in test
	assert.Equal(t, 2, len(spans))

	// Verify attributes are correctly added to the spans. Omit the one local span
	for _, span := range spans[0 : len(spans)-1] {

		switch span.Name {
		case cassQueryName:
			assert.Equal(t, insertStmt, span.Attributes[cassStatementKey].AsString())
			assert.Equal(t, parentSpan.SpanContext().SpanID.String(), span.ParentSpanID.String())
		default:
			t.Fatalf("unexpected span name %s", span.Name)
		}
		assert.NotNil(t, span.Attributes[cassVersionKey].AsString())
		assert.Equal(t, cluster.Hosts[0], span.Attributes[cassHostKey].AsString())
		assert.Equal(t, int32(cluster.Port), span.Attributes[cassPortKey].AsInt32())
		assert.Equal(t, "up", strings.ToLower(span.Attributes[cassHostStateKey].AsString()))
	}

	// Check metrics
	controller.Stop()

	assert.Equal(t, 3, len(exporter.records))
	expected := []testRecord{
		{
			Name:      "cassandra.queries",
			MeterName: "github.com/gocql/gocql",
			Labels: []kv.KeyValue{
				cassHostID("test-id"),
				cassKeyspace(keyspace),
				cassStatement(insertStmt),
			},
			Number: 1,
		},
		{
			Name:      "cassandra.rows",
			MeterName: "github.com/gocql/gocql",
			Labels: []kv.KeyValue{
				cassHostID("test-id"),
				cassKeyspace(keyspace),
			},
			Number: 0,
		},
		{
			Name:      "cassandra.latency",
			MeterName: "github.com/gocql/gocql",
			Labels: []kv.KeyValue{
				cassHostID("test-id"),
				cassKeyspace(keyspace),
			},
		},
	}

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		agg := record.Aggregation()
		switch name {
		case "cassandra.queries":
			recordEqual(t, expected[0], record)
			numberEqual(t, expected[0].Number, agg)
		case "cassandra.rows":
			recordEqual(t, expected[1], record)
			numberEqual(t, expected[1].Number, agg)
		case "cassandra.latency":
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
	tracer := mocktracer.NewTracer("gocql-test")

	ctx, parentSpan := tracer.Start(context.Background(), "gocql-test")

	session, err := NewSessionWithTracing(
		ctx,
		cluster,
		WithTracer(tracer),
		WithConnectInstrumentation(false),
	)
	assert.NoError(t, err)
	defer session.Close()

	batch := session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	ids := make([]gocql.UUID, 10)
	stmts := make([]string, 10)
	for i := 0; i < 10; i++ {
		ids[i] = gocql.TimeUUID()
		title := fmt.Sprintf("batch-title-%d", i)
		stmts[i] = fmt.Sprintf("insert into %s (id, title) values (?, ?)", tableName)
		batch.Query(stmts[i], ids[i], title)
	}

	err = session.ExecuteBatch(batch)
	assert.NoError(t, err)

	parentSpan.End()

	spans := tracer.EndedSpans()
	// total spans:
	// 1 span for the query
	// 1 span for the local span
	assert.Equal(t, 2, len(spans))
	span := spans[0]

	assert.Equal(t, cassBatchQueryName, span.Name)
	assert.Equal(t, parentSpan.SpanContext().SpanID, span.ParentSpanID)
	assert.NotNil(t, span.Attributes[cassVersionKey].AsString())
	assert.Equal(t, cluster.Hosts[0], span.Attributes[cassHostKey].AsString())
	assert.Equal(t, int32(cluster.Port), span.Attributes[cassPortKey].AsInt32())
	assert.Equal(t, "up", strings.ToLower(span.Attributes[cassHostStateKey].AsString()))
	assert.Equal(t, int32(len(stmts)), span.Attributes[cassBatchQueriesKey].AsInt32())

	controller.Stop()

	// Check metrics
	assert.Equal(t, 2, len(exporter.records))
	expected := []testRecord{
		{
			Name:      "cassandra.batch.queries",
			MeterName: "github.com/gocql/gocql",
			Labels: []kv.KeyValue{
				cassHostID("test-id"),
				cassKeyspace(keyspace),
			},
			Number: 1,
		},
		{
			Name:      "cassandra.latency",
			MeterName: "github.com/gocql/gocql",
			Labels: []kv.KeyValue{
				cassHostID("test-id"),
				cassKeyspace(keyspace),
			},
		},
	}

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		agg := record.Aggregation()
		switch name {
		case "cassandra.batch.queries":
			recordEqual(t, expected[0], record)
			numberEqual(t, expected[0].Number, agg)
		case "cassandra.latency":
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
	tracer := mocktracer.NewTracer("gocql-test")
	connectObserver := &mockConnectObserver{0}

	session, err := NewSessionWithTracing(
		context.Background(),
		cluster,
		WithTracer(tracer),
		WithConnectObserver(connectObserver),
	)
	assert.NoError(t, err)
	defer session.Close()

	spans := tracer.EndedSpans()

	assert.Less(t, 0, connectObserver.callCount)

	controller.Stop()

	// Verify the span attributes
	for _, span := range spans {
		assert.Equal(t, cassConnectName, span.Name)
		assert.NotNil(t, span.Attributes[cassVersionKey].AsString())
		assert.Equal(t, cluster.Hosts[0], span.Attributes[cassHostKey].AsString())
		assert.Equal(t, int32(cluster.Port), span.Attributes[cassPortKey].AsInt32())
		assert.Equal(t, "up", strings.ToLower(span.Attributes[cassHostStateKey].AsString()))
	}

	// Verify the metrics
	expected := []testRecord{
		{
			Name:      "cassandra.connections",
			MeterName: "github.com/gocql/gocql",
			Labels: []kv.KeyValue{
				cassHost("127.0.0.1"),
				cassHostID("test-id"),
			},
		},
	}

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		switch name {
		case "cassandra.connections":
			recordEqual(t, expected[0], record)
		default:
			t.Fatalf("wrong metric %s", name)
		}
	}
}

func TestGetHost(t *testing.T) {
	hostAndPort := "localhost:9042"
	assert.Equal(t, "localhost", getHost(hostAndPort))

	hostAndPort = "127.0.0.1:9042"
	assert.Equal(t, "127.0.0.1", getHost(hostAndPort))

	hostAndPort = ":9042"
	assert.Equal(t, "", getHost(hostAndPort))
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

// beforeEach creates a metric export pipeline with mockExporter
// to enable testing metric collection.
func getController(t *testing.T) *push.Controller {
	controller := mockExportPipeline(t)
	InstrumentWithProvider(controller.Provider())
	return controller
}

// recordEqual checks that the given metric name and instrumentation names are equal.
func recordEqual(t *testing.T, expected testRecord, actual export.Record) {
	descriptor := actual.Descriptor()
	assert.Equal(t, expected.Name, descriptor.Name())
	assert.Equal(t, expected.MeterName, descriptor.InstrumentationName())
	for _, label := range expected.Labels {
		actualValue, ok := actual.Labels().Value(label.Key)
		assert.True(t, ok)
		assert.NotNil(t, actualValue)
		// Can't test equality of host id
		if label.Key != cassHostIDKey {
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

// beforeAll recreates the testing keyspace so that a new table
// can be created. This allows the test to be run multiple times
// consecutively withouth any issues arising.
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

// afterEach removes the keyspace from the database for later test sessions.
func afterEach() {
	cluster := gocql.NewCluster("localhost")
	cluster.Consistency = gocql.LocalQuorum
	cluster.Keyspace = keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("failed to connect to database during afterEach, %v", err)
	}
	if err = session.Query(fmt.Sprintf("truncate table %s", tableName)).Exec(); err != nil {
		log.Fatalf("failed to truncate table, %v", err)
	}
}

func TestMain(m *testing.M) {
	if _, present := os.LookupEnv("INTEGRATION"); !present {
		fmt.Println("--- SKIP: to enable integration test, set the INTEGRATION environment variable")
		os.Exit(0)
	}
	beforeAll()
	os.Exit(m.Run())
}

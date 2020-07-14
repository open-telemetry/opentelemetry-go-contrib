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
	connectObserver := &mockConnectObserver{}

	session, err := NewSessionWithTracing(
		cluster,
		WithTracer(tracer),
		WithConnectObserver(connectObserver),
	)
	assert.NoError(t, err)
	defer session.Close()

	ctx, parentSpan := tracer.Start(context.Background(), "gocql-test")

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
	numberOfConnections := connectObserver.callCount
	// there should be numberOfConnections + 1 Query + 1 Batch spans
	assert.Equal(t, numberOfConnections+2, len(spans))
	assert.Greater(t, numberOfConnections, 0, "at least one connection needs to have been made")

	// Verify attributes are correctly added to the spans. Omit the one local span
	for _, span := range spans[0 : len(spans)-1] {

		switch span.Name {
		case cassQueryName:
			assert.Equal(t, insertStmt, span.Attributes[CassStatementKey].AsString())
			assert.Equal(t, parentSpan.SpanContext().SpanID.String(), span.ParentSpanID.String())
			break
		case cassConnectName:
			numberOfConnections--
		default:
			t.Fatalf("unexpected span name %s", span.Name)
		}
		assert.NotNil(t, span.Attributes[CassVersionKey].AsString())
		assert.Equal(t, cluster.Hosts[0], span.Attributes[CassHostKey].AsString())
		assert.Equal(t, int32(cluster.Port), span.Attributes[CassPortKey].AsInt32())
		assert.Equal(t, "up", strings.ToLower(span.Attributes[CassHostStateKey].AsString()))
	}
	assert.Equal(t, 0, numberOfConnections)

	// Check metrics
	controller.Stop()

	assert.Equal(t, 3, len(exporter.records))
	expected := []testRecord{
		testRecord{
			Name:      "cassandra.connections",
			MeterName: "github.com/gocql/gocql",
			// TODO: Labels
			Number: 3,
		},
		testRecord{
			Name:      "cassandra.queries",
			MeterName: "github.com/gocql/gocql",
			// TODO: Labels
			Number: 1,
		},
		testRecord{
			Name:      "cassandra.rows",
			MeterName: "github.com/gocql/gocql",
			Number:    0,
		},
	}

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		agg := record.Aggregation()
		switch name {
		case "cassandra.connections":
			recordEqual(t, expected[0], record)
			numberEqual(t, expected[0].Number, agg)
		case "cassandra.queries":
			recordEqual(t, expected[1], record)
			numberEqual(t, expected[1].Number, agg)
			break
		case "cassandra.rows":
			recordEqual(t, expected[2], record)
			numberEqual(t, expected[2].Number, agg)
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

	session, err := NewSessionWithTracing(
		cluster,
		WithTracer(tracer),
		WithConnectInstrumentation(false),
	)
	assert.NoError(t, err)
	defer session.Close()

	ctx, parentSpan := tracer.Start(context.Background(), "gocql-test")

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
	assert.Equal(t, 2, len(spans))
	span := spans[0]

	assert.Equal(t, cassBatchQueryName, span.Name)
	assert.Equal(t, parentSpan.SpanContext().SpanID, span.ParentSpanID)
	assert.NotNil(t, span.Attributes[CassVersionKey].AsString())
	assert.Equal(t, cluster.Hosts[0], span.Attributes[CassHostKey].AsString())
	assert.Equal(t, int32(cluster.Port), span.Attributes[CassPortKey].AsInt32())
	assert.Equal(t, "up", strings.ToLower(span.Attributes[CassHostStateKey].AsString()))
	assert.Equal(t, stmts, span.Attributes[CassBatchStatementsKey].AsArray())

	controller.Stop()

	assert.Equal(t, 1, len(exporter.records))
	expected := []testRecord{
		testRecord{
			Name:      "cassandra.batch_queries",
			MeterName: "github.com/gocql/gocql",
			// TODO: Labels
			Number: 1,
		},
	}

	for _, record := range exporter.records {
		name := record.Descriptor().Name()
		agg := record.Aggregation()
		switch name {
		case "cassandra.batch_queries":
			recordEqual(t, expected[0], record)
			numberEqual(t, expected[0].Number, agg)
		default:
			t.Fatalf("wrong metric %s", name)
		}
	}

}

func TestConnection(t *testing.T) {
	defer afterEach()
	cluster := getCluster()
	tracer := mocktracer.NewTracer("gocql-test")

	session, err := NewSessionWithTracing(cluster, WithTracer(tracer))
	assert.NoError(t, err)
	defer session.Close()

	spans := tracer.EndedSpans()

	for _, span := range spans {
		assert.Equal(t, cassConnectName, span.Name)
		assert.NotNil(t, span.Attributes[CassVersionKey].AsString())
		assert.Equal(t, cluster.Hosts[0], span.Attributes[CassHostKey].AsString())
		assert.Equal(t, int32(cluster.Port), span.Attributes[CassPortKey].AsInt32())
		assert.Equal(t, "up", strings.ToLower(span.Attributes[CassHostStateKey].AsString()))
	}
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
		log.Print("--- SKIP: to enable integration test, set the INTEGRATION environment variable")
		os.Exit(0)
	}
	beforeAll()
	os.Exit(m.Run())
}

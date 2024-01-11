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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/contrib/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-mongo-driver")
	os.Exit(m.Run())
}

type validator func(sdktrace.ReadOnlySpan) bool

func TestDBCrudOperation(t *testing.T) {
	commonValidators := []validator{
		func(s sdktrace.ReadOnlySpan) bool {
			return assert.Equal(t, "test-collection.insert", s.Name(), "expected %s", s.Name())
		},
		func(s sdktrace.ReadOnlySpan) bool {
			return assert.Contains(t, s.Attributes(), attribute.String("db.operation", "insert"))
		},
		func(s sdktrace.ReadOnlySpan) bool {
			return assert.Contains(t, s.Attributes(), attribute.String("db.mongodb.collection", "test-collection"))
		},
		func(s sdktrace.ReadOnlySpan) bool {
			return assert.Equal(t, codes.Unset, s.Status().Code)
		},
	}

	tt := []struct {
		title          string
		operation      func(context.Context, *mongo.Database) (interface{}, error)
		excludeCommand bool
		validators     []validator
	}{
		{
			title: "insert",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
			},
			excludeCommand: false,
			validators: append(commonValidators, func(s sdktrace.ReadOnlySpan) bool {
				for _, attr := range s.Attributes() {
					if attr.Key == "db.statement" {
						return assert.Contains(t, attr.Value.AsString(), `"test-item":"test-value"`)
					}
				}
				return false
			}),
		},
		{
			title: "insert",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
			},
			excludeCommand: true,
			validators: append(commonValidators, func(s sdktrace.ReadOnlySpan) bool {
				for _, attr := range s.Attributes() {
					if attr.Key == "db.statement" {
						return false
					}
				}
				return true
			}),
		},
	}
	for _, tc := range tt {
		title := tc.title
		if tc.excludeCommand {
			title = title + "/excludeCommand"
		} else {
			title = title + "/includeCommand"
		}
		t.Run(title, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			mr := sdkmetric.NewManualReader()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer("test").Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			opts.Monitor = otelmongo.NewMonitor(
				otelmongo.WithTracerProvider(provider),
				otelmongo.WithCommandAttributeDisabled(tc.excludeCommand),
				otelmongo.WithMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(mr))),
			)
			opts.ApplyURI(addr)
			client, err := mongo.Connect(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			_, err = tc.operation(ctx, client.Database("test-database"))
			if err != nil {
				t.Error(err)
			}

			span.End()

			spans := sr.Ended()
			if !assert.Len(t, spans, 2, "expected 2 spans, received %d", len(spans)) {
				t.FailNow()
			}
			assert.Len(t, spans, 2)
			assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
			assert.Equal(t, spans[0].Parent().SpanID(), spans[1].SpanContext().SpanID())
			assert.Equal(t, span.SpanContext().SpanID(), spans[1].SpanContext().SpanID())

			s := spans[0]
			assert.Equal(t, trace.SpanKindClient, s.SpanKind())
			attrs := s.Attributes()
			assert.Contains(t, attrs, attribute.String("db.system", "mongodb"))
			assert.Contains(t, attrs, attribute.String("net.peer.name", "localhost"))
			assert.Contains(t, attrs, attribute.Int64("net.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("net.transport", "ip_tcp"))
			assert.Contains(t, attrs, attribute.String("db.name", "test-database"))
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}

			var md metricdata.ResourceMetrics
			err = mr.Collect(context.Background(), &md)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Len(t, md.ScopeMetrics, 1)
			scopedMetrics := md.ScopeMetrics[0]
			assert.Equal(t, instrumentation.Scope{
				Name:      otelmongo.ScopeName,
				Version:   otelmongo.Version(),
				SchemaURL: "",
			}, scopedMetrics.Scope)
			assert.Len(t, scopedMetrics.Metrics, 1)
			metrics := scopedMetrics.Metrics[0]
			assert.Equal(t, "command.duration", metrics.Name)
			assert.Equal(t, "Duration of finished commands", metrics.Description)
			assert.Equal(t, "ms", metrics.Unit)
			assert.Len(t, metrics.Data.(metricdata.Histogram[float64]).DataPoints, 1)
			dp := metrics.Data.(metricdata.Histogram[float64]).DataPoints[0]
			attrs = dp.Attributes.ToSlice()
			assert.Contains(t, attrs, attribute.String("db.system", "mongodb"))
			assert.Contains(t, attrs, attribute.String("db.operation", "insert"))
			assert.Contains(t, attrs, attribute.String("db.name", "test-database"))
			assert.Contains(t, attrs, attribute.String("net.peer.name", "localhost"))
			assert.Contains(t, attrs, attribute.Int64("net.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("net.transport", "ip_tcp"))
			assert.Contains(t, attrs, attribute.String("db.mongodb.collection", "test-collection"))
			assert.Contains(t, attrs, attribute.String("otel.status_code", "OK"))
			assert.EqualValues(t, 1, dp.Count)
			assert.Greater(t, dp.Sum, 0.0)
		})
	}
}

func TestDBCollectionAttribute(t *testing.T) {
	tt := []struct {
		title      string
		operation  func(context.Context, *mongo.Database) (interface{}, error)
		validators []validator
	}{
		{
			title: "delete",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").DeleteOne(ctx, bson.D{{Key: "test-item"}})
			},
			validators: []validator{
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, "test-collection.delete", s.Name())
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Contains(t, s.Attributes(), attribute.String("db.operation", "delete"))
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Contains(t, s.Attributes(), attribute.String("db.mongodb.collection", "test-collection"))
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, codes.Unset, s.Status().Code)
				},
			},
		},
		{
			title: "listCollectionNames",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.ListCollectionNames(ctx, bson.D{})
			},
			validators: []validator{
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, "listCollections", s.Name())
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Contains(t, s.Attributes(), attribute.String("db.operation", "listCollections"))
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, codes.Unset, s.Status().Code)
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.title, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer("test").Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			opts.Monitor = otelmongo.NewMonitor(
				otelmongo.WithTracerProvider(provider),
				otelmongo.WithCommandAttributeDisabled(true),
				otelmongo.WithMeterProvider(sdkmetric.NewMeterProvider()),
			)
			opts.ApplyURI(addr)
			client, err := mongo.Connect(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			_, err = tc.operation(ctx, client.Database("test-database"))
			if err != nil {
				t.Error(err)
			}

			span.End()

			spans := sr.Ended()
			if !assert.Len(t, spans, 2, "expected 2 spans, received %d", len(spans)) {
				t.FailNow()
			}
			assert.Len(t, spans, 2)
			assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
			assert.Equal(t, spans[0].Parent().SpanID(), spans[1].SpanContext().SpanID())
			assert.Equal(t, span.SpanContext().SpanID(), spans[1].SpanContext().SpanID())

			s := spans[0]
			assert.Equal(t, trace.SpanKindClient, s.SpanKind())
			attrs := s.Attributes()
			assert.Contains(t, attrs, attribute.String("db.system", "mongodb"))
			assert.Contains(t, attrs, attribute.String("net.peer.name", "localhost"))
			assert.Contains(t, attrs, attribute.Int64("net.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("net.transport", "ip_tcp"))
			assert.Contains(t, attrs, attribute.String("db.name", "test-database"))
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}
		})
	}
}

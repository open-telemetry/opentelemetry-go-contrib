// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo" // nolint:staticcheck  // deprecated.
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

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
		mockResponses  []bson.D
		excludeCommand bool
		validators     []validator
	}{
		{
			title: "insert",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
			},
			mockResponses:  []bson.D{{{"ok", 1}}},
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
			mockResponses:  []bson.D{{{"ok", 1}}},
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

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer("test").Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			opts.Monitor = otelmongo.NewMonitor(
				otelmongo.WithTracerProvider(provider),
				otelmongo.WithCommandAttributeDisabled(tc.excludeCommand),
			)

			opts.ApplyURI(addr)

			mock := newMockDeployment()
			opts.Deployment = mock

			client, err := mongo.Connect(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			mock.addResponses(tc.mockResponses...)
			t.Cleanup(mock.clearResponses)

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
			assert.Contains(t, attrs, attribute.String("net.peer.name", "<mock_connection>"))
			assert.Contains(t, attrs, attribute.Int64("net.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("net.transport", "ip_tcp"))
			assert.Contains(t, attrs, attribute.String("db.name", "test-database"))
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}
		})
	}
}

func TestDBCollectionAttribute(t *testing.T) {
	tt := []struct {
		title         string
		operation     func(context.Context, *mongo.Database) (interface{}, error)
		mockResponses []bson.D
		validators    []validator
	}{
		{
			title: "delete",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").DeleteOne(ctx, bson.D{{Key: "test-item"}})
			},
			mockResponses: []bson.D{{{"ok", 1}}},
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
			mockResponses: []bson.D{{{"ok", 1}, {"cursor", bson.D{{"firstBatch", bson.A{}}}}}},
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
			)

			opts.ApplyURI(addr)

			mock := newMockDeployment()
			opts.Deployment = mock

			client, err := mongo.Connect(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			mock.addResponses(tc.mockResponses...)
			t.Cleanup(mock.clearResponses)

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
			assert.Contains(t, attrs, attribute.String("net.peer.name", "<mock_connection>"))
			assert.Contains(t, attrs, attribute.Int64("net.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("net.transport", "ip_tcp"))
			assert.Contains(t, attrs, attribute.String("db.name", "test-database"))
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}
		})
	}
}

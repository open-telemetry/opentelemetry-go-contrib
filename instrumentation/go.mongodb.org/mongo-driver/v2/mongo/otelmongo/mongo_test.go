// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"

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
			return assert.Contains(t, s.Attributes(), attribute.String("db.operation.name", "insert"))
		},
		func(s sdktrace.ReadOnlySpan) bool {
			return assert.Contains(t, s.Attributes(), attribute.String("db.collection.name", "test-collection"))
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
			mockResponses:  []bson.D{{{Key: "ok", Value: 1}}},
			excludeCommand: false,
			validators: append(commonValidators, func(s sdktrace.ReadOnlySpan) bool {
				for _, attr := range s.Attributes() {
					if attr.Key == "db.query.text" {
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
			mockResponses:  []bson.D{{{Key: "ok", Value: 1}}},
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
		tc := tc

		title := tc.title
		if tc.excludeCommand {
			title = title + "/excludeCommand"
		} else {
			title = title + "/includeCommand"
		}

		t.Run(title, func(t *testing.T) {
			md := drivertest.NewMockDeployment()

			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer("test").Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			//nolint:staticcheck
			opts.Deployment = md // This method is the current documented way to set the mongodb mock. See https://github.com/mongodb/mongo-go-driver/blob/v2.0.0/x/mongo/driver/drivertest/opmsg_deployment_test.go#L24
			opts.Monitor = NewMonitor(
				WithTracerProvider(provider),
				WithCommandAttributeDisabled(tc.excludeCommand),
			)
			opts.ApplyURI(addr)

			md.AddResponses(tc.mockResponses...)
			client, err := mongo.Connect(opts)
			defer func() {
				err := client.Disconnect(context.Background())
				require.NoError(t, err)
			}()
			require.NoError(t, err)

			_, err = tc.operation(ctx, client.Database("test-database"))
			require.NoError(t, err)

			span.End()

			spans := sr.Ended()
			require.Len(t, spans, 2, "expected 2 spans, received %d", len(spans))
			assert.Len(t, spans, 2)
			assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
			assert.Equal(t, spans[0].Parent().SpanID(), spans[1].SpanContext().SpanID())
			assert.Equal(t, span.SpanContext().SpanID(), spans[1].SpanContext().SpanID())

			s := spans[0]
			assert.Equal(t, trace.SpanKindClient, s.SpanKind())
			attrs := s.Attributes()
			assert.Contains(t, attrs, attribute.String("db.system.name", "mongodb"))
			assert.Contains(t, attrs, attribute.String("network.peer.address", "<mock_connection>"))
			assert.Contains(t, attrs, attribute.Int64("network.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("network.transport", "tcp"))
			assert.Contains(t, attrs, attribute.String("db.namespace", "test-database"))
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
			mockResponses: []bson.D{{{Key: "ok", Value: 1}}},
			validators: []validator{
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, "test-collection.delete", s.Name())
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Contains(t, s.Attributes(), attribute.String("db.operation.name", "delete"))
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Contains(t, s.Attributes(), attribute.String("db.collection.name", "test-collection"))
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
			mockResponses: []bson.D{
				{
					{Key: "ok", Value: 1},
					{Key: "cursor", Value: bson.D{{Key: "firstBatch", Value: bson.A{}}}},
				},
			},
			validators: []validator{
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, "listCollections", s.Name())
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Contains(t, s.Attributes(), attribute.String("db.operation.name", "listCollections"))
				},
				func(s sdktrace.ReadOnlySpan) bool {
					return assert.Equal(t, codes.Unset, s.Status().Code)
				},
			},
		},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.title, func(t *testing.T) {
			md := drivertest.NewMockDeployment()

			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer("test").Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			//nolint:staticcheck
			opts.Deployment = md // This method is the current documented way to set the mongodb mock. See https://github.com/mongodb/mongo-go-driver/blob/v2.0.0/x/mongo/driver/drivertest/opmsg_deployment_test.go#L24
			opts.Monitor = NewMonitor(
				WithTracerProvider(provider),
				WithCommandAttributeDisabled(true),
			)
			opts.ApplyURI(addr)

			md.AddResponses(tc.mockResponses...)
			client, err := mongo.Connect(opts)
			require.NoError(t, err)

			defer func() {
				err := client.Disconnect(context.Background())
				require.NoError(t, err)
			}()

			_, err = tc.operation(ctx, client.Database("test-database"))
			require.NoError(t, err)

			span.End()

			spans := sr.Ended()
			require.Len(t, spans, 2, "expected 2 spans, received %d", len(spans))
			assert.Len(t, spans, 2)
			assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
			assert.Equal(t, spans[0].Parent().SpanID(), spans[1].SpanContext().SpanID())
			assert.Equal(t, span.SpanContext().SpanID(), spans[1].SpanContext().SpanID())

			s := spans[0]
			assert.Equal(t, trace.SpanKindClient, s.SpanKind())
			attrs := s.Attributes()
			assert.Contains(t, attrs, attribute.String("db.system.name", "mongodb"))
			assert.Contains(t, attrs, attribute.String("network.peer.address", "<mock_connection>"))
			assert.Contains(t, attrs, attribute.Int64("network.peer.port", int64(27017)))
			assert.Contains(t, attrs, attribute.String("network.transport", "tcp"))
			assert.Contains(t, attrs, attribute.String("db.namespace", "test-database"))
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}
		})
	}
}

func TestPeerInfo(t *testing.T) {
	tests := []struct {
		name         string
		connectionID string
		expectedHost string
		expectedPort int
	}{
		{
			name:         "IPv4 with port",
			connectionID: "127.0.0.1:27018",
			expectedHost: "127.0.0.1",
			expectedPort: 27018,
		},
		{
			name:         "IPv4 without port",
			connectionID: "127.0.0.1",
			expectedHost: "127.0.0.1",
			expectedPort: 27017,
		},
		{
			name:         "IPv6 with port",
			connectionID: "[::1]:27019",
			expectedHost: "::1",
			expectedPort: 27019,
		},
		{
			name:         "IPv6 without port with square brackets",
			connectionID: "[::1]",
			expectedHost: "[::1]",
			expectedPort: 27017,
		},
		{
			name:         "IPv6 without port",
			connectionID: "::1",
			expectedHost: "::1",
			expectedPort: 27017,
		},
		{
			name:         "Hostname with port",
			connectionID: "example.com:27020",
			expectedHost: "example.com",
			expectedPort: 27020,
		},
		{
			name:         "Hostname without port",
			connectionID: "example.com",
			expectedHost: "example.com",
			expectedPort: 27017,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			evt := &event.CommandStartedEvent{
				ConnectionID: tc.connectionID,
			}
			host, port := peerInfo(evt)
			assert.Equal(t, tc.expectedHost, host)
			assert.Equal(t, tc.expectedPort, port)
		})
	}
}

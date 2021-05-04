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

package otelmongo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/internal/util"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-mongo-driver")
	os.Exit(m.Run())
}

type validator func(*oteltest.Span) bool

func TestDBCrudOperation(t *testing.T) {
	commonValidators := []validator{
		func(s *oteltest.Span) bool {
			return assert.Equal(t, "test-collection.insert", s.Name())
		},
		func(s *oteltest.Span) bool {
			return assert.Equal(t,"insert", s.Attributes()["db.operation"].AsString())
		},
		func(s *oteltest.Span) bool {
			return assert.Equal(t, "test-collection", s.Attributes()["db.mongodb.collection"].AsString())
		},
		func(s *oteltest.Span) bool {
			return assert.Equal(t, codes.Unset, s.StatusCode())
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
			validators: append(commonValidators, func(s *oteltest.Span) bool {
				return assert.Contains(t, s.Attributes()["db.statement"].AsString(), `"test-item":"test-value"`)
			}),
		},
		{
			title: "insert",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
			},
			excludeCommand: true,
			validators: append(commonValidators, func(s *oteltest.Span) bool {
				return assert.NotContains(t, s.Attributes()["db.statement"].AsString(), `"test-item":"test-value"`)
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
			sr := new(oteltest.SpanRecorder)
			provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer(defaultTracerName).Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			opts.Monitor = NewMonitor(WithTracerProvider(provider), WithCommandAttributeDisabled(tc.excludeCommand))
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

			spans := sr.Completed()
			if !assert.Len(t, spans, 2, "expected 2 spans, received %d", len(spans)) {
				t.FailNow()
			}
			assert.Len(t, spans, 2)
			assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
			assert.Equal(t, spans[0].ParentSpanID(), spans[1].SpanContext().SpanID())
			assert.Equal(t, span.SpanContext().SpanID(), spans[1].SpanContext().SpanID())

			s := spans[0]
			assert.Equal(t, trace.SpanKindClient, s.SpanKind())
			assert.Equal(t, "mongodb", s.Attributes()["db.system"].AsString())
			assert.Equal(t, "localhost", s.Attributes()["net.peer.name"].AsString())
			assert.Equal(t, int64(27017), s.Attributes()["net.peer.port"].AsInt64())
			assert.Equal(t, "IP.TCP", s.Attributes()["net.transport"].AsString())
			assert.Equal(t, "test-database", s.Attributes()["db.name"].AsString())
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}
		})
	}

}
func TestDBCollectionAttribute(t *testing.T) {
	tt := []struct {
		title          string
		operation      func(context.Context, *mongo.Database) (interface{}, error)
		validators     []validator
	}{
		{
			title: "delete",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").DeleteOne(ctx, bson.D{{Key: "test-item"}})
			},
			validators: []validator{
				func(s *oteltest.Span) bool {
					return assert.Equal(t, "test-collection.delete", s.Name())
				},
				func(s *oteltest.Span) bool {
					return assert.Equal(t, "delete", s.Attributes()["db.operation"].AsString())
				},
				func(s *oteltest.Span) bool {
					return assert.Equal(t, "test-collection", s.Attributes()["db.mongodb.collection"].AsString())
				},
				func(s *oteltest.Span) bool {
					return assert.Equal(t, codes.Unset, s.StatusCode())
				},
			},
		},
		{
			title: "listCollectionNames",
			operation: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.ListCollectionNames(ctx, bson.D{})
			},
			validators: []validator{
				func(s *oteltest.Span) bool {
					return assert.Equal(t, "listCollections", s.Name())
				},
				func(s *oteltest.Span) bool {
					return assert.Equal(t, "listCollections", s.Attributes()["db.operation"].AsString())
				},
				func(s *oteltest.Span) bool {
					return assert.Equal(t, codes.Unset, s.StatusCode())
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.title, func(t *testing.T) {
			sr := new(oteltest.SpanRecorder)
			provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer(defaultTracerName).Start(ctx, "mongodb-test")

			addr := "mongodb://localhost:27017/?connect=direct"
			opts := options.Client()
			opts.Monitor = NewMonitor(WithTracerProvider(provider), WithCommandAttributeDisabled(true))
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

			spans := sr.Completed()
			if !assert.Len(t, spans, 2, "expected 2 spans, received %d", len(spans)) {
				t.FailNow()
			}
			assert.Len(t, spans, 2)
			assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
			assert.Equal(t, spans[0].ParentSpanID(), spans[1].SpanContext().SpanID())
			assert.Equal(t, span.SpanContext().SpanID(), spans[1].SpanContext().SpanID())

			s := spans[0]
			assert.Equal(t, trace.SpanKindClient, s.SpanKind())
			assert.Equal(t, "mongodb", s.Attributes()["db.system"].AsString())
			assert.Equal(t, "localhost", s.Attributes()["net.peer.name"].AsString())
			assert.Equal(t, int64(27017), s.Attributes()["net.peer.port"].AsInt64())
			assert.Equal(t, "IP.TCP", s.Attributes()["net.transport"].AsString())
			assert.Equal(t, "test-database", s.Attributes()["db.name"].AsString())
			for _, v := range tc.validators {
				assert.True(t, v(s))
			}

		})
	}
}

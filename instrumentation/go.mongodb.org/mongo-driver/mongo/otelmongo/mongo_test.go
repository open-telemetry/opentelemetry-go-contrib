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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-mongo-driver")
	os.Exit(m.Run())
}

type validator struct {
	expected interface{}
	accessor func(*oteltest.Span) interface{}
}

func TestDBOperation(t *testing.T) {
	tt := []struct {
		title      string
		op         func(context.Context, *mongo.Database) (interface{}, error)
		validators []validator
	}{
		{
			title: "insert",
			op: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
			},
			validators: []validator{
				{"test-collection.insert", func(s *oteltest.Span) interface{} { return s.Name() }},
				{"insert", func(s *oteltest.Span) interface{} { return s.Attributes()["db.operation"].AsString() }},
				{"test-collection", func(s *oteltest.Span) interface{} { return s.Attributes()["db.mongodb.collection"].AsString() }},
			},
		},
		{
			title: "delete",
			op: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.Collection("test-collection").DeleteOne(ctx, bson.D{{Key: "test-item"}})
			},
			validators: []validator{
				{"test-collection.delete", func(s *oteltest.Span) interface{} { return s.Name() }},
				{"delete", func(s *oteltest.Span) interface{} { return s.Attributes()["db.operation"].AsString() }},
				{"test-collection", func(s *oteltest.Span) interface{} { return s.Attributes()["db.mongodb.collection"].AsString() }},
			},
		},
		{
			title: "listCollectionNames",
			op: func(ctx context.Context, db *mongo.Database) (interface{}, error) {
				return db.ListCollectionNames(ctx, bson.D{})
			},
			validators: []validator{
				{"listCollections", func(s *oteltest.Span) interface{} { return s.Name() }},
				{"listCollections", func(s *oteltest.Span) interface{} { return s.Attributes()["db.operation"].AsString() }},
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
			opts.Monitor = NewMonitor(WithTracerProvider(provider), WithCommandAttributeDisabled(tc.commandAttributeDisabled))
			opts.ApplyURI(addr)
			client, err := mongo.Connect(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			_, err = tc.op(ctx, client.Database("test-database"))
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

			s := spans[0]
			assert.Equal(t, "mongodb", s.Attributes()["db.system"].AsString())
			assert.Equal(t, "localhost", s.Attributes()["net.peer.name"].AsString())
			assert.Equal(t, int64(27017), s.Attributes()["net.peer.port"].AsInt64())
			assert.Equal(t, "IP.TCP", s.Attributes()["net.transport"].AsString())
			assert.Equal(t, "test-database", s.Attributes()["db.name"].AsString())
			if tc.commandAttributeDisabled {
				assert.NotContains(t, s.Attributes()[DBStatementKey].AsString(), `"test-item":"test-value"`)
			} else {
				assert.Contains(t, s.Attributes()[DBStatementKey].AsString(), `"test-item":"test-value"`)
			}
			for _, v := range tc.validators {
				assert.Equal(t, v.expected, v.accessor(s))
			}
			assert.Equal(t, codes.Unset, s.StatusCode())

		})
	}
}

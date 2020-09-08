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

package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/contrib/internal/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-mongo-driver")
	os.Exit(m.Run())
}

func Test(t *testing.T) {
	provider, mt := mocktracer.NewProviderAndTracer(defaultTracerName)

	hostname, port := "localhost", "27017"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	ctx, span := mt.Start(ctx, "mongodb-test")

	addr := "mongodb://localhost:27017/?connect=direct"
	opts := options.Client()
	opts.Monitor = NewMonitor("mongo", WithTracerProvider(provider))
	opts.ApplyURI(addr)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Database("test-database").Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
	if err != nil {
		t.Fatal(err)
	}

	span.End()

	spans := mt.EndedSpans()
	assert.Len(t, spans, 2)
	assert.Equal(t, spans[0].SpanContext().TraceID, spans[1].SpanContext().TraceID)

	s := spans[0]
	assert.Equal(t, "mongo", s.Attributes[ServiceNameKey].AsString())
	assert.Equal(t, "mongo.insert", s.Attributes[ResourceNameKey].AsString())
	assert.Equal(t, hostname, s.Attributes[PeerHostnameKey].AsString())
	assert.Equal(t, port, s.Attributes[PeerPortKey].AsString())
	assert.Contains(t, s.Attributes[DBStatementKey].AsString(), `"test-item":"test-value"`)
	assert.Equal(t, "test-database", s.Attributes[DBInstanceKey].AsString())
	assert.Equal(t, "mongo", s.Attributes[DBTypeKey].AsString())
}

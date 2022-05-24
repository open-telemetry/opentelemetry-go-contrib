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
	"os"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache"
	"go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache/internal"
	"go.opentelemetry.io/contrib/internal/util"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-gomemcache")
	os.Exit(m.Run())
}

func TestOperation(t *testing.T) {
	c, sr := initClientWithSpanRecorder(t)

	mi := &memcache.Item{
		Key:   "foo",
		Value: []byte("bar"),
	}
	err := c.Add(mi)
	require.NoError(t, err)

	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, oteltrace.SpanKindClient, spans[0].SpanKind())
	assert.Equal(t, string(internal.OperationAdd), spans[0].Name())
	assert.Len(t, spans[0].Attributes(), 3)

	attrs := spans[0].Attributes()
	assert.Contains(t, attrs, internal.MemcacheDBSystem())
	assert.Contains(t, attrs, internal.MemcacheDBOperation(internal.OperationAdd))
	assert.Contains(t, attrs, internal.MemcacheDBItemKeyName.String(mi.Key))
}

func TestOperationWithCacheMissError(t *testing.T) {
	key := "foo"
	c, sr := initClientWithSpanRecorder(t)

	_, err := c.Get(key)
	assert.Error(t, err)

	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, oteltrace.SpanKindClient, spans[0].SpanKind())
	assert.Equal(t, string(internal.OperationGet), spans[0].Name())
	assert.Len(t, spans[0].Attributes(), 3)

	attrs := spans[0].Attributes()
	assert.Contains(t, attrs, internal.MemcacheDBSystem())
	assert.Contains(t, attrs, internal.MemcacheDBOperation(internal.OperationGet))
	assert.Contains(t, attrs, internal.MemcacheDBItemKeyName.String(key))

	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Equal(t, err.Error(), spans[0].Status().Description)
}

// tests require running memcached instance.
func initClientWithSpanRecorder(t *testing.T) (*otelmemcache.Client, *tracetest.SpanRecorder) {
	host, port := "localhost", "11211"

	mc := memcache.New(host + ":" + port)
	require.NoError(t, clearDB(mc))

	sr := tracetest.NewSpanRecorder()
	c := otelmemcache.NewClientWithTracing(
		mc,
		otelmemcache.WithTracerProvider(
			trace.NewTracerProvider(trace.WithSpanProcessor(sr)),
		),
	)

	return c, sr
}

func clearDB(c *memcache.Client) error {
	return c.DeleteAll()
}

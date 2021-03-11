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

package otelmemcache

import (
	"os"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-gomemcache")
	os.Exit(m.Run())
}

func TestNewClientWithTracing(t *testing.T) {
	c := NewClientWithTracing(
		memcache.New(),
	)

	assert.NotNil(t, c.Client)
	assert.NotNil(t, c.tracer)
}

func TestOperation(t *testing.T) {
	c, sr := initClientWithSpanRecorder(t)

	mi := &memcache.Item{
		Key:   "foo",
		Value: []byte("bar"),
	}
	err := c.Add(mi)
	require.NoError(t, err)

	spans := sr.Completed()
	assert.Len(t, spans, 1)
	assert.Equal(t, oteltrace.SpanKindClient, spans[0].SpanKind())
	assert.Equal(t, string(operationAdd), spans[0].Name())
	assert.Len(t, spans[0].Attributes(), 3)

	expectedAttributeMap := map[attribute.Key]attribute.Value{
		memcacheDBSystem().Key:                                  memcacheDBSystem().Value,
		memcacheDBOperation(operationAdd).Key:                   memcacheDBOperation(operationAdd).Value,
		attribute.Key(memcacheDBItemKeyName).String(mi.Key).Key: attribute.Key(memcacheDBItemKeyName).String(mi.Key).Value,
	}
	assert.Equal(t, expectedAttributeMap, spans[0].Attributes())
}

func TestOperationWithCacheMissError(t *testing.T) {
	key := "foo"
	c, sr := initClientWithSpanRecorder(t)

	_, err := c.Get(key)
	assert.Error(t, err)

	spans := sr.Completed()
	assert.Len(t, spans, 1)
	assert.Equal(t, oteltrace.SpanKindClient, spans[0].SpanKind())
	assert.Equal(t, string(operationGet), spans[0].Name())
	assert.Len(t, spans[0].Attributes(), 3)

	expectedAttributeMap := map[attribute.Key]attribute.Value{
		memcacheDBSystem().Key:                               memcacheDBSystem().Value,
		memcacheDBOperation(operationGet).Key:                memcacheDBOperation(operationGet).Value,
		attribute.Key(memcacheDBItemKeyName).String(key).Key: attribute.Key(memcacheDBItemKeyName).String(key).Value,
	}
	assert.Equal(t, expectedAttributeMap, spans[0].Attributes())

	assert.Equal(t, codes.Error, spans[0].StatusCode())
	assert.Equal(t, err.Error(), spans[0].StatusMessage())
}

// tests require running memcached instance
func initClientWithSpanRecorder(t *testing.T) (*Client, *oteltest.SpanRecorder) {
	host, port := "localhost", "11211"

	mc := memcache.New(host + ":" + port)
	require.NoError(t, clearDB(mc))

	sr := new(oteltest.SpanRecorder)
	c := NewClientWithTracing(
		mc,
		WithTracerProvider(
			oteltest.NewTracerProvider(
				oteltest.WithSpanRecorder(sr),
			),
		),
	)

	return c, sr
}

func clearDB(c *memcache.Client) error {
	return c.DeleteAll()
}

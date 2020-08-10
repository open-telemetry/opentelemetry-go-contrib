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

package gomemcache

import (
	"fmt"
	"os"
	"testing"

	"go.opentelemetry.io/otel/api/standard"

	"google.golang.org/grpc/codes"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/otel/api/kv"
	oteltrace "go.opentelemetry.io/otel/api/trace"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_, ok := os.LookupEnv("INTEGRATION")
	if !ok {
		fmt.Println("--- SKIP: to enable integration test, set the INTEGRATION environment variable")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestNewClientWithTracing(t *testing.T) {
	c := NewClientWithTracing(
		memcache.New(),
	)

	assert.NotNil(t, c.Client)
	assert.NotNil(t, c.cfg)
	assert.NotEqual(t, &oteltrace.NoopProvider{}, c.cfg.tracer)
	assert.Equal(t, defaultServiceName, c.cfg.serviceName)
}

func TestOperation(t *testing.T) {
	c, mt := initClientWithMockTracer(t)

	mi := &memcache.Item{
		Key:   "foo",
		Value: []byte("bar"),
	}
	err := c.Add(mi)
	require.NoError(t, err)

	spans := mt.EndedSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, oteltrace.SpanKindClient, spans[0].Kind)
	assert.Equal(t, operationAdd, spans[0].Name)
	assert.Len(t, spans[0].Attributes, 4)

	expectedKvMap := map[kv.Key]kv.Value{
		standard.ServiceNameKey:                             standard.ServiceNameKey.String(defaultServiceName).Value,
		memcacheDBSystem().Key:                              memcacheDBSystem().Value,
		memcacheDBOperation(operationAdd).Key:               memcacheDBOperation(operationAdd).Value,
		kv.Key(memcacheDBItemKeyKeyName).String(mi.Key).Key: kv.Key(memcacheDBItemKeyKeyName).String(mi.Key).Value,
	}
	assert.Equal(t, expectedKvMap, spans[0].Attributes)
}

func TestOperationWithCacheMissError(t *testing.T) {
	key := "foo"
	c, mt := initClientWithMockTracer(t)

	_, err := c.Get(key)
	assert.Error(t, err)

	spans := mt.EndedSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, oteltrace.SpanKindClient, spans[0].Kind)
	assert.Equal(t, operationGet, spans[0].Name)
	assert.Len(t, spans[0].Attributes, 4)

	expectedKvMap := map[kv.Key]kv.Value{
		standard.ServiceNameKey:                          standard.ServiceNameKey.String(defaultServiceName).Value,
		memcacheDBSystem().Key:                           memcacheDBSystem().Value,
		memcacheDBOperation(operationGet).Key:            memcacheDBOperation(operationGet).Value,
		kv.Key(memcacheDBItemKeyKeyName).String(key).Key: kv.Key(memcacheDBItemKeyKeyName).String(key).Value,
	}
	assert.Equal(t, expectedKvMap, spans[0].Attributes)

	assert.Equal(t, codes.NotFound, spans[0].Status)
	assert.Equal(t, err.Error(), spans[0].StatusMessage)
}

// tests require running memcached instance
func initClientWithMockTracer(t *testing.T) (*Client, *mocktracer.Tracer) {
	mt := mocktracer.NewTracer("memcache-test")

	host, port := "localhost", "11211"

	mc := memcache.New(host + ":" + port)
	require.NoError(t, clearDB(mc))

	c := NewClientWithTracing(
		mc,
		WithTracer(mt),
	)

	return c, mt
}

func clearDB(c *memcache.Client) error {
	return c.DeleteAll()
}

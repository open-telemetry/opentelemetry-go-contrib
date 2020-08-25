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
	"context"

	"github.com/bradfitz/gomemcache/memcache"

	"go.opentelemetry.io/contrib"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

const (
	defaultTracerName  = "go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache"
	defaultServiceName = "memcached"
)

// Client is a wrapper around *memcache.Client.
type Client struct {
	*memcache.Client
	cfg    *config
	tracer oteltrace.Tracer
	ctx    context.Context
}

// NewClientWithTracing wraps the provided memcache client to allow
// tracing of all client operations. Accepts options to set trace provider
// and service name, otherwise uses registered global trace provider and
// default value for service name.
//
// Every client operation starts a span with appropriate attributes,
// executes the operation and ends the span (additionally also sets a status
// error code and message, if an error occurs). Optionally, client context can
// be set before an operation with the WithContext method.
func NewClientWithTracing(client *memcache.Client, opts ...Option) *Client {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.serviceName == "" {
		cfg.serviceName = defaultServiceName
	}

	if cfg.traceProvider == nil {
		cfg.traceProvider = otelglobal.TraceProvider()
	}

	return &Client{
		client,
		cfg,
		cfg.traceProvider.Tracer(
			defaultTracerName,
			oteltrace.WithInstrumentationVersion(contrib.SemVersion()),
		),
		context.Background(),
	}
}

// attrsByOperationAndItemKey returns appropriate span attributes on the basis
// of the operation name and item key(s) (if available)
func (c *Client) attrsByOperationAndItemKey(operation operation, key ...string) []label.KeyValue {
	labels := []label.KeyValue{
		semconv.ServiceNameKey.String(c.cfg.serviceName),
		memcacheDBSystem(),
		memcacheDBOperation(operation),
	}

	if len(key) > 0 {
		labels = append(labels, memcacheDBItemKeys(key...))
	}

	return labels
}

// Starts span with appropriate span kind and attributes
func (c *Client) startSpan(operationName operation, itemKey ...string) oteltrace.Span {
	opts := []oteltrace.StartOption{
		// for database client calls, always use CLIENT span kind
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			c.attrsByOperationAndItemKey(operationName, itemKey...)...,
		),
	}

	_, span := c.tracer.Start(
		c.ctx,
		string(operationName),
		opts...,
	)

	return span
}

// Ends span and, if applicable, sets error status
func endSpan(s oteltrace.Span, err error) {
	if err != nil {
		s.SetStatus(memcacheErrToStatusCode(err), err.Error())
	}
	s.End()
}

// WithContext retruns a copy of the client with provided context
func (c *Client) WithContext(ctx context.Context) *Client {
	cc := c.Client
	return &Client{
		Client: cc,
		cfg:    c.cfg,
		tracer: c.tracer,
		ctx:    ctx,
	}
}

// Add invokes the add operation and traces it
func (c *Client) Add(item *memcache.Item) error {
	s := c.startSpan(operationAdd, item.Key)
	err := c.Client.Add(item)
	endSpan(s, err)
	return err
}

// CompareAndSwap invokes the compare-and-swap operation and traces it
func (c *Client) CompareAndSwap(item *memcache.Item) error {
	s := c.startSpan(operationCompareAndSwap, item.Key)
	err := c.Client.CompareAndSwap(item)
	endSpan(s, err)
	return err
}

// Decrement invokes the decrement operation and traces it
func (c *Client) Decrement(key string, delta uint64) (uint64, error) {
	s := c.startSpan(operationDecrement, key)
	newValue, err := c.Client.Decrement(key, delta)
	endSpan(s, err)
	return newValue, err
}

// Delete invokes the delete operation and traces it
func (c *Client) Delete(key string) error {
	s := c.startSpan(operationDelete, key)
	err := c.Client.Delete(key)
	endSpan(s, err)
	return err
}

// DeleteAll invokes the delete all operation and traces it
func (c *Client) DeleteAll() error {
	s := c.startSpan(operationDeleteAll)
	err := c.Client.DeleteAll()
	endSpan(s, err)
	return err
}

// FlushAll invokes the flush all operation and traces it
func (c *Client) FlushAll() error {
	s := c.startSpan(operationFlushAll)
	err := c.Client.FlushAll()
	endSpan(s, err)
	return err
}

// Get invokes the get operation and traces it
func (c *Client) Get(key string) (*memcache.Item, error) {
	s := c.startSpan(operationGet, key)
	item, err := c.Client.Get(key)
	endSpan(s, err)
	return item, err
}

// GetMulti invokes the get operation for multiple keys and traces it
func (c *Client) GetMulti(keys []string) (map[string]*memcache.Item, error) {
	s := c.startSpan(operationGet, keys...)
	items, err := c.Client.GetMulti(keys)
	endSpan(s, err)
	return items, err
}

// Increment invokes the increment operation and traces it
func (c *Client) Increment(key string, delta uint64) (uint64, error) {
	s := c.startSpan(operationIncrement, key)
	newValue, err := c.Client.Increment(key, delta)
	endSpan(s, err)
	return newValue, err
}

// Ping invokes the ping operation and traces it
func (c *Client) Ping() error {
	s := c.startSpan(operationPing)
	err := c.Client.Ping()
	endSpan(s, err)
	return err
}

// Replace invokes the replace operation and traces it
func (c *Client) Replace(item *memcache.Item) error {
	s := c.startSpan(operationReplace, item.Key)
	err := c.Client.Replace(item)
	endSpan(s, err)
	return err
}

// Set invokes the set operation and traces it
func (c *Client) Set(item *memcache.Item) error {
	s := c.startSpan(operationSet, item.Key)
	err := c.Client.Set(item)
	endSpan(s, err)
	return err
}

// Touch invokes the touch operation and traces it
func (c *Client) Touch(key string, seconds int32) error {
	s := c.startSpan(operationTouch, key)
	err := c.Client.Touch(key, seconds)
	endSpan(s, err)
	return err
}

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

package otelkit

import (
	"context"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	operationKey = contextKey("operation")
)

// compile time assertion
var _ endpoint.Failer = failedResponse{}

type customError struct {
	message string
}

func (e customError) Error() string {
	return e.message
}

type failedResponse struct {
	err error
}

func (r failedResponse) Failed() error { return r.err }

func passEndpoint(_ context.Context, req interface{}) (interface{}, error) {
	if err, _ := req.(error); err != nil {
		return nil, err
	}
	return req, nil
}

func TestEndpointMiddleware(t *testing.T) {
	t.Run("GlobalTracer", func(t *testing.T) {
		otel.SetTracerProvider(oteltest.NewTracerProvider())

		mw := EndpointMiddleware()

		e := func(ctx context.Context, _ interface{}) (interface{}, error) {
			span := trace.SpanFromContext(ctx)
			_, ok := span.(*oteltest.Span)
			assert.True(t, ok)
			spanTracer := span.Tracer()
			mockTracer, ok := spanTracer.(*oteltest.Tracer)
			require.True(t, ok)
			assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit", mockTracer.Name)

			return nil, nil
		}

		_, _ = mw(e)(context.Background(), nil)
	})

	t.Run("DefaultOperationAndAttributes", func(t *testing.T) {
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		mw := EndpointMiddleware(
			WithTracerProvider(provider),
			WithOperation("operation"),
			WithAttributes(attribute.String("key", "value")),
		)

		_, _ = mw(passEndpoint)(context.Background(), nil)

		spans := sr.Completed()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "operation", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.StatusCode())
		assert.Equal(t, attribute.StringValue("value"), span.Attributes()["key"])
	})

	t.Run("OperationAndAttributesFromContext", func(t *testing.T) {
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		mw := EndpointMiddleware(
			WithTracerProvider(provider),
			WithOperationGetter(func(ctx context.Context, name string) string {
				operation, _ := ctx.Value(operationKey).(string)

				return operation
			}),
			WithAttributeGetter(func(ctx context.Context) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("key", "value"),
				}
			}),
		)

		ctx := context.WithValue(context.Background(), operationKey, "operation")

		_, _ = mw(passEndpoint)(ctx, nil)

		spans := sr.Completed()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "operation", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.StatusCode())
		assert.Equal(t, attribute.StringValue("value"), span.Attributes()["key"])
	})

	t.Run("Overrides", func(t *testing.T) {
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		mw := EndpointMiddleware(
			WithTracerProvider(provider),
			WithOperation("operations"),
			WithOperationGetter(func(ctx context.Context, name string) string {
				operation, _ := ctx.Value(operationKey).(string)

				return operation
			}),
			WithAttributes(attribute.String("key", "value")),
			WithAttributeGetter(func(ctx context.Context) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("key2", "value2"),
				}
			}),
		)

		ctx := context.WithValue(context.Background(), operationKey, "other_operation")

		_, _ = mw(passEndpoint)(ctx, nil)

		spans := sr.Completed()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "other_operation", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.StatusCode())
		assert.Equal(t, attribute.StringValue("value"), span.Attributes()["key"])
		assert.Equal(t, attribute.StringValue("value2"), span.Attributes()["key2"])
	})

	t.Run("Error", func(t *testing.T) {
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		mw := EndpointMiddleware(
			WithTracerProvider(provider),
		)

		ctx := context.Background()

		_, _ = mw(func(_ context.Context, req interface{}) (interface{}, error) {
			return nil, customError{"something went wrong"}
		})(ctx, nil)

		spans := sr.Completed()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, defaultSpanName, span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Error, span.StatusCode())

		events := span.Events()
		require.Len(t, events, 1)

		assert.Equal(t, "error", events[0].Name)
		assert.Equal(t, attribute.StringValue("go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit.customError"), events[0].Attributes["error.type"])
		assert.Equal(t, attribute.StringValue("something went wrong"), events[0].Attributes["error.message"])
	})

	t.Run("BusinessError", func(t *testing.T) {
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		mw := EndpointMiddleware(
			WithTracerProvider(provider),
		)

		ctx := context.Background()

		_, _ = mw(func(_ context.Context, req interface{}) (interface{}, error) {
			return failedResponse{err: customError{"some business error"}}, nil
		})(ctx, nil)

		spans := sr.Completed()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, defaultSpanName, span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Error, span.StatusCode())

		events := span.Events()
		require.Len(t, events, 1)

		assert.Equal(t, "error", events[0].Name)
		assert.Equal(t, attribute.StringValue("go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit.customError"), events[0].Attributes["error.type"])
		assert.Equal(t, attribute.StringValue("some business error"), events[0].Attributes["error.message"])
	})

	t.Run("IgnoredBusinessError", func(t *testing.T) {
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		mw := EndpointMiddleware(
			WithTracerProvider(provider),
			WithIgnoreBusinessError(true),
		)

		ctx := context.Background()

		_, _ = mw(func(_ context.Context, req interface{}) (interface{}, error) {
			return failedResponse{err: customError{"some business error"}}, nil
		})(ctx, nil)

		spans := sr.Completed()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, defaultSpanName, span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.StatusCode())

		events := span.Events()
		require.Len(t, events, 1)

		assert.Equal(t, "error", events[0].Name)
		assert.Equal(t, attribute.StringValue("go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit.customError"), events[0].Attributes["error.type"])
		assert.Equal(t, attribute.StringValue("some business error"), events[0].Attributes["error.message"])
	})
}

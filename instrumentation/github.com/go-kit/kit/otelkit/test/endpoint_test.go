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
	"context"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	operationKey = contextKey("operation")
)

// compile time assertion.
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
		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

		mw := otelkit.EndpointMiddleware()

		e := func(ctx context.Context, _ interface{}) (interface{}, error) {
			return nil, nil
		}

		_, _ = mw(e)(context.Background(), nil)
		assert.Len(t, sr.Ended(), 1)
	})

	t.Run("DefaultOperationAndAttributes", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		mw := otelkit.EndpointMiddleware(
			otelkit.WithTracerProvider(provider),
			otelkit.WithOperation("operation"),
			otelkit.WithAttributes(attribute.String("key", "value")),
		)

		_, _ = mw(passEndpoint)(context.Background(), nil)

		spans := sr.Ended()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "operation", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.Status().Code)
		assert.Contains(t, span.Attributes(), attribute.String("key", "value"))
	})

	t.Run("OperationAndAttributesFromContext", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		mw := otelkit.EndpointMiddleware(
			otelkit.WithTracerProvider(provider),
			otelkit.WithOperationGetter(func(ctx context.Context, name string) string {
				operation, _ := ctx.Value(operationKey).(string)

				return operation
			}),
			otelkit.WithAttributeGetter(func(ctx context.Context) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("key", "value"),
				}
			}),
		)

		ctx := context.WithValue(context.Background(), operationKey, "operation")

		_, _ = mw(passEndpoint)(ctx, nil)

		spans := sr.Ended()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "operation", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.Status().Code)
		assert.Contains(t, span.Attributes(), attribute.String("key", "value"))
	})

	t.Run("Overrides", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		mw := otelkit.EndpointMiddleware(
			otelkit.WithTracerProvider(provider),
			otelkit.WithOperation("operations"),
			otelkit.WithOperationGetter(func(ctx context.Context, name string) string {
				operation, _ := ctx.Value(operationKey).(string)

				return operation
			}),
			otelkit.WithAttributes(attribute.String("key", "value")),
			otelkit.WithAttributeGetter(func(ctx context.Context) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("key2", "value2"),
				}
			}),
		)

		ctx := context.WithValue(context.Background(), operationKey, "other_operation")

		_, _ = mw(passEndpoint)(ctx, nil)

		spans := sr.Ended()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "other_operation", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.Status().Code)
		assert.Contains(t, span.Attributes(), attribute.String("key", "value"))
		assert.Contains(t, span.Attributes(), attribute.String("key2", "value2"))
	})

	t.Run("Error", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		mw := otelkit.EndpointMiddleware(
			otelkit.WithTracerProvider(provider),
		)

		ctx := context.Background()

		_, _ = mw(func(_ context.Context, req interface{}) (interface{}, error) {
			return nil, customError{"something went wrong"}
		})(ctx, nil)

		spans := sr.Ended()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "gokit/endpoint", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Error, span.Status().Code)

		events := span.Events()
		require.Len(t, events, 1)

		assert.Equal(t, "exception", events[0].Name)
		assert.Contains(t, events[0].Attributes, attribute.String("exception.type", "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/test.customError"))
		assert.Contains(t, events[0].Attributes, attribute.String("exception.message", "something went wrong"))
	})

	t.Run("BusinessError", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		mw := otelkit.EndpointMiddleware(
			otelkit.WithTracerProvider(provider),
		)

		ctx := context.Background()

		_, _ = mw(func(_ context.Context, req interface{}) (interface{}, error) {
			return failedResponse{err: customError{"some business error"}}, nil
		})(ctx, nil)

		spans := sr.Ended()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "gokit/endpoint", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Error, span.Status().Code)

		events := span.Events()
		require.Len(t, events, 1)

		assert.Equal(t, "exception", events[0].Name)
		assert.Contains(t, events[0].Attributes, attribute.String("exception.type", "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/test.customError"))
		assert.Contains(t, events[0].Attributes, attribute.String("exception.message", "some business error"))
	})

	t.Run("IgnoredBusinessError", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		mw := otelkit.EndpointMiddleware(
			otelkit.WithTracerProvider(provider),
			otelkit.WithIgnoreBusinessError(true),
		)

		ctx := context.Background()

		_, _ = mw(func(_ context.Context, req interface{}) (interface{}, error) {
			return failedResponse{err: customError{"some business error"}}, nil
		})(ctx, nil)

		spans := sr.Ended()
		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "gokit/endpoint", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Equal(t, codes.Unset, span.Status().Code)

		events := span.Events()
		require.Len(t, events, 1)

		assert.Equal(t, "exception", events[0].Name)
		assert.Contains(t, events[0].Attributes, attribute.String("exception.type", "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/test.customError"))
		assert.Contains(t, events[0].Attributes, attribute.String("exception.message", "some business error"))
	})
}

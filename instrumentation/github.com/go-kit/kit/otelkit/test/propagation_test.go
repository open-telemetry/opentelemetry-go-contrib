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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// W3c Trace Context
// https://www.w3.org/TR/trace-context/
// trace-version "-" trace-id "-" parent-id "-" trace-flags
const (
	emptyParentID = "00000000000000000000000000000000"
	parentID      = "0af7651916cd43dd8448eb211c80319c" // trace-id, but parent of our span being tested
	traceParent   = "00-" + parentID + "-b9c7c989f97918e1-01"
)

func TestGrpcPropagationMiddleware(t *testing.T) {
	t.Run("WithParentTrace", func(t *testing.T) {

		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		ctx := context.Background()

		// Add metadata WITH traceparent as a instrumentrament gRPC client would
		md := metadata.Pairs(
			"traceparent", traceParent,
		)
		ctx = metadata.NewIncomingContext(ctx, md)

		prop := otelkit.GrpcPropagationMiddleware()
		mw := otelkit.EndpointMiddleware()

		e := func(ctx context.Context, _ interface{}) (interface{}, error) {
			return nil, nil
		}

		_, _ = prop(mw(e))(ctx, nil)
		spans := sr.Ended()

		assert.Len(t, spans, 1)
		assert.Equal(t, spans[0].Parent().HasTraceID(), true)
		assert.Equal(t, spans[0].Parent().TraceID().String(), parentID)
	})

	t.Run("WithoutParentTrace", func(t *testing.T) {

		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		ctx := context.Background()

		// Add metadata WITHOUT traceparent as a instrumented gRPC client would
		md := metadata.Pairs(
			"traceparent", "",
		)
		ctx = metadata.NewIncomingContext(ctx, md)

		prop := otelkit.GrpcPropagationMiddleware()
		mw := otelkit.EndpointMiddleware()

		e := func(ctx context.Context, _ interface{}) (interface{}, error) {
			return nil, nil
		}

		_, _ = prop(mw(e))(ctx, nil)
		spans := sr.Ended()

		assert.Len(t, spans, 1)
		assert.Equal(t, spans[0].Parent().HasTraceID(), false)
		assert.Equal(t, spans[0].Parent().TraceID().String(), emptyParentID)
	})

}

func TestHTTPPropagationMiddleware(t *testing.T) {
	t.Run("WithParentTrace", func(t *testing.T) {

		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		ctx := context.Background()

		// Add request WITH traceparent as a instrumented HTTP client would
		r, _ := http.NewRequest("", "", nil)
		r.Header.Set("traceparent", traceParent)

		prop := otelkit.HTTPPropagationMiddleware()
		mw := otelkit.EndpointMiddleware()

		e := func(ctx context.Context, _ interface{}) (interface{}, error) {
			return nil, nil
		}

		_, _ = prop(mw(e))(ctx, r)
		spans := sr.Ended()

		assert.Len(t, spans, 1)
		assert.Equal(t, spans[0].Parent().HasTraceID(), true)
		assert.Equal(t, spans[0].Parent().TraceID().String(), parentID)
	})

	t.Run("WithoutParentTrace", func(t *testing.T) {

		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		ctx := context.Background()

		// Add request WITHOUT traceparent as a instrumentramented HTTP client would
		r, _ := http.NewRequest("", "", nil)
		r.Header.Set("traceparent", "")

		prop := otelkit.HTTPPropagationMiddleware()
		mw := otelkit.EndpointMiddleware()

		e := func(ctx context.Context, _ interface{}) (interface{}, error) {
			return nil, nil
		}

		_, _ = prop(mw(e))(ctx, r)
		spans := sr.Ended()

		assert.Len(t, spans, 1)
		assert.Equal(t, spans[0].Parent().HasTraceID(), false)
		assert.Equal(t, spans[0].Parent().TraceID().String(), emptyParentID)
	})

}

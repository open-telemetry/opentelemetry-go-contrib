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

package otelmacaron

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/macaron.v1"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGetSpanNotInstrumented(t *testing.T) {
	m := macaron.Classic()
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test-tracer")
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(noop.NewTracerProvider())

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x01},
		SpanID:  oteltrace.SpanID{0x01},
	})
	ctx = oteltrace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = tracer.Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	m := macaron.Classic()
	m.Use(Middleware("foobar"))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	m.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test-tracer")
	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x01},
		SpanID:  oteltrace.SpanID{0x01},
	})
	ctx = oteltrace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = tracer.Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracerProvider(tp), WithPropagators(b3)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	m.ServeHTTP(w, r)
}

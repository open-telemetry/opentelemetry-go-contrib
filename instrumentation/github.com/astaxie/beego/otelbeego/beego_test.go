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

package otelbeego

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"

	"github.com/astaxie/beego"
	beegoCtx "github.com/astaxie/beego/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const middleWareName = "test-router"

func replaceBeego() {
	beego.BeeApp = beego.NewApp()
}

func ctxTest() (context.Context, func(*testing.T, context.Context)) {
	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)

	return ctx, func(t *testing.T, ctx context.Context) {
		got := trace.SpanContextFromContext(ctx)
		assert.Equal(t, sc.TraceID(), got.TraceID())
		assert.Equal(t, sc.SpanID(), got.SpanID())
		assert.Equal(t, sc.TraceFlags(), got.TraceFlags())
		assert.Equal(t, sc.TraceState(), got.TraceState())
		assert.Equal(t, sc.IsRemote(), got.IsRemote())
	}
}

func TestSpanFromContextDefaultProvider(t *testing.T) {
	defer replaceBeego()
	provider := metric.NewNoopMeterProvider()
	global.SetMeterProvider(provider)
	otel.SetTracerProvider(trace.NewNoopTracerProvider())

	ctx, eval := ctxTest()
	router := beego.NewControllerRegister()
	router.Get("/hello-with-span", func(ctx *beegoCtx.Context) {
		eval(t, ctx.Request.Context())
		ctx.ResponseWriter.WriteHeader(http.StatusAccepted)
	})

	rr := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/hello-with-span", nil)
	require.NoError(t, err)

	mw := NewOTelBeegoMiddleWare(middleWareName)

	mw(router).ServeHTTP(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Result().StatusCode)
}

func TestSpanFromContextCustomProvider(t *testing.T) {
	defer replaceBeego()
	provider := metric.NewNoopMeterProvider()
	ctx, eval := ctxTest()
	router := beego.NewControllerRegister()
	router.Get("/hello-with-span", func(ctx *beegoCtx.Context) {
		eval(t, ctx.Request.Context())
		ctx.ResponseWriter.WriteHeader(http.StatusAccepted)
	})

	rr := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/hello-with-span", nil)
	require.NoError(t, err)

	mw := NewOTelBeegoMiddleWare(
		middleWareName,
		WithTracerProvider(trace.NewNoopTracerProvider()),
		WithMeterProvider(provider),
	)

	mw(router).ServeHTTP(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Result().StatusCode)
}

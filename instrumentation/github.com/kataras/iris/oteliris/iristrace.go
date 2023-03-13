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

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/kataras/iris/gintrace.go

package oteliris // import "go.opentelemetry.io/contrib/instrumentation/github.com/kataras/iris/oteliris"

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/semconv/v1.18.0/httpconv"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerKey  = "otel-go-contrib-tracer"
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/kataras/iris/oteliris"
)

// Middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func Middleware(service string, opts ...Option) iris.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(SemVersion()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	return func(ctx iris.Context) {
		for _, f := range cfg.Filters {
			if !f(ctx.Request()) {
				// Serve the request to the next middleware
				// if a filter rejects the request.
				ctx.Next()
				return
			}
		}
		ctx.Values().Set(tracerKey, tracer)
		savedCtx := ctx.Request().Context()
		defer func() {
			ctx.ResetRequest(ctx.Request().WithContext(savedCtx))
		}()
		tracerCtx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(ctx.Request().Header))
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(httpconv.ServerRequest(service, ctx.Request())...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}
		var spanName string
		if cfg.SpanNameFormatter == nil {
			spanName = ctx.GetCurrentRoute().Path()
		} else {
			spanName = cfg.SpanNameFormatter(ctx.Request())
		}
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", ctx.Method())
		} else {
			rAttr := semconv.HTTPRoute(spanName)
			opts = append(opts, oteltrace.WithAttributes(rAttr))
		}
		tracerCtx, span := tracer.Start(tracerCtx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		ctx.ResetRequest(ctx.Request().WithContext(tracerCtx))

		// serve the request to the next middleware
		ctx.Next()

		status := ctx.GetStatusCode()
		span.SetStatus(httpconv.ServerStatus(status))
		if status > 0 {
			span.SetAttributes(semconv.HTTPStatusCode(status))
		}

		if ctxErr := ctx.GetErr(); ctxErr != nil {
			span.SetAttributes(attribute.String("iris.errors", ctxErr.Error()))
		}
	}
}

// HTML will trace the rendering of the template as a child of the
// span in the given context. This is a replacement for
// iris.Context.View function - it invokes the original function after
// setting up the span.
func HTML(ctx iris.Context, code int, name string, obj interface{}) {
	var tracer oteltrace.Tracer
	tracerInterface := ctx.Values().Get(tracerKey)
	if tracerInterface != nil {
		var ok bool
		if tracer, ok = tracerInterface.(oteltrace.Tracer); !ok {
			tracerInterface = nil
		}
	}
	if tracerInterface == nil {
		tracer = otel.GetTracerProvider().Tracer(
			tracerName,
			oteltrace.WithInstrumentationVersion(SemVersion()),
		)
	}

	savedContext := ctx.Request().Context()
	defer func() {
		ctx.ResetRequest(ctx.Request().WithContext(savedContext))
	}()
	opt := oteltrace.WithAttributes(attribute.String("go.template", name))
	_, span := tracer.Start(savedContext, "iris.renderer.html", opt)
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("error rendering template:%s: %s", name, r)
			span.RecordError(err)
			span.SetStatus(codes.Error, "template failure")
			span.End()
			panic(r)
		} else {
			span.End()
		}
	}()
	ctx.StatusCode(code)
	ctx.View(name, obj)
}

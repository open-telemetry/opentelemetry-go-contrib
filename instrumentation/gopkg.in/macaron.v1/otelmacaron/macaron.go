// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmacaron // import "go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron"

import (
	"fmt"
	"net/http"

	"gopkg.in/macaron.v1"

	"go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron/internal/semconvutil"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron"

// Middleware returns a macaron Handler to trace requests to the server.
func Middleware(service string, opts ...Option) macaron.Handler {
	cfg := newConfig(opts)
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		oteltrace.WithInstrumentationVersion(Version()),
	)
	return func(res http.ResponseWriter, req *http.Request, c *macaron.Context) {
		savedCtx := c.Req.Request.Context()
		defer func() {
			c.Req.Request = c.Req.Request.WithContext(savedCtx)
		}()

		ctx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(c.Req.Header))
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(semconvutil.HTTPServerRequest(service, c.Req.Request)...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}
		// TODO: span name should be router template not the actual request path, eg /user/:id vs /user/123
		spanName := c.Req.RequestURI
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Req.Method)
		}
		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		c.Req.Request = c.Req.Request.WithContext(ctx)

		// serve the request to the next middleware
		c.Next()

		status := c.Resp.Status()
		span.SetStatus(semconvutil.HTTPServerStatus(status))
		if status > 0 {
			span.SetAttributes(semconv.HTTPStatusCode(status))
		}
	}
}

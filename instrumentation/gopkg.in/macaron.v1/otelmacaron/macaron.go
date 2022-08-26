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

package otelmacaron // import "go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron"

import (
	"fmt"
	"net/http"

	"gopkg.in/macaron.v1"

	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// instrumentationName is the name of this instrumentation package.
const instrumentationName = "go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron"

// Middleware returns a macaron Handler to trace requests to the server.
func Middleware(service string, opts ...Option) macaron.Handler {
	cfg := newConfig(opts)
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		oteltrace.WithInstrumentationVersion(SemVersion()),
	)
	return func(res http.ResponseWriter, req *http.Request, c *macaron.Context) {
		savedCtx := c.Req.Request.Context()
		defer func() {
			c.Req.Request = c.Req.Request.WithContext(savedCtx)
		}()

		ctx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(c.Req.Header))
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Req.Request)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Req.Request)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, "", c.Req.Request)...),
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
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(status, oteltrace.SpanKindServer)
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}
}

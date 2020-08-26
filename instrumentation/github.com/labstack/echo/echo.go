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

package echo

import (
	"fmt"

	"github.com/labstack/echo/v4"

	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

const (
	tracerKey  = "otel-go-contrib-tracer-labstack-echo"
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo"
)

// Middleware returns echo middleware which will trace incoming requests.
func Middleware(service string, opts ...Option) echo.MiddlewareFunc {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Tracer == nil {
		cfg.Tracer = otelglobal.Tracer(tracerName)
	}
	if cfg.Propagators == nil {
		cfg.Propagators = otelglobal.Propagators()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(tracerKey, cfg.Tracer)
			request := c.Request()
			savedCtx := request.Context()
			defer func() {
				request = request.WithContext(savedCtx)
				c.SetRequest(request)
			}()
			ctx := otelpropagation.ExtractHTTP(savedCtx, cfg.Propagators, request.Header)
			opts := []oteltrace.StartOption{
				oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", request)...),
				oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(request)...),
				oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, c.Path(), request)...),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			}
			spanName := c.Path()
			if spanName == "" {
				spanName = fmt.Sprintf("HTTP %s route not found", request.Method)
			}

			ctx, span := cfg.Tracer.Start(ctx, spanName, opts...)
			defer span.End()

			// pass the span through the request context
			c.SetRequest(request.WithContext(ctx))

			// serve the request to the next middleware
			err := next(c)
			if err != nil {
				span.SetAttributes(label.String("echo.error", err.Error()))
				// invokes the registered HTTP error handler
				c.Error(err)
			}

			attrs := semconv.HTTPAttributesFromHTTPStatusCode(c.Response().Status)
			spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(c.Response().Status)
			span.SetAttributes(attrs...)
			span.SetStatus(spanStatus, spanMessage)

			return err
		}
	}
}

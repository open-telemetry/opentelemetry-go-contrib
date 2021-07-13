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

package otelfiber

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"

	otelcontrib "go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerKey  = "otel-go-contrib-tracer-gofiber-fiber"
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/otelfiber"
)

// Middleware returns fiber handler which will trace incoming requests.
func Middleware(service string, opts ...Option) fiber.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}

	return func(c *fiber.Ctx) error {
		c.Locals(tracerKey, tracer)
		savedCtx, cancel := context.WithCancel(c.Context())

		defer func() {
			c.SetUserContext(savedCtx)
			cancel()
		}()

		reqHeader := make(http.Header)
		c.Request().Header.VisitAll(func(k, v []byte) {
			sk := string(k)
			sv := string(v)

			switch sk {
			case fiber.HeaderTransferEncoding:
				reqHeader[fiber.HeaderTransferEncoding] = append(reqHeader[fiber.HeaderTransferEncoding], sv)
			default:
				reqHeader[sk] = []string{sv}
			}
		})

		ctx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(reqHeader))
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(semconv.HTTPServerNameKey.String(service),
				semconv.HTTPMethodKey.String(c.Method()),
				semconv.HTTPTargetKey.String(string(c.Request().RequestURI())),
				semconv.HTTPURLKey.String(c.OriginalURL()),
				semconv.NetHostIPKey.String(c.IP()),
				semconv.HTTPUserAgentKey.String(string(c.Request().Header.UserAgent())),
				semconv.HTTPRequestContentLengthKey.Int(c.Request().Header.ContentLength()),
				semconv.HTTPSchemeKey.String(c.Protocol()),
				semconv.HTTPClientIPKey.String(c.IP()),
				semconv.NetTransportTCP),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}
		// temporary set to c.Path() first
		// update with c.Route().Path after c.Next() is called
		// to get pathRaw
		spanName := c.Path()
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Path())
		}

		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through userContext
		c.SetUserContext(ctx)

		// serve the request to the next middleware
		err := c.Next()

		span.SetName(c.Route().Path)
		span.SetAttributes(semconv.HTTPRouteKey.String(c.Route().Path))

		if err != nil {
			span.SetAttributes(attribute.String("fiber.error", err.Error()))
			// invokes the registered HTTP error handler
			_ = c.App().Config().ErrorHandler(c, err)
		}

		attrs := semconv.HTTPAttributesFromHTTPStatusCode(c.Response().StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(c.Response().StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)

		return nil
	}
}

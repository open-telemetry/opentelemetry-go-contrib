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

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace.go

package otelgin // import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin/internal/semconvutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerKey           = "otel-go-contrib-tracer"
	instrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func Middleware(service string, opts ...Option) gin.HandlerFunc {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		oteltrace.WithInstrumentationVersion(Version()),
	)
	meter := cfg.MeterProvider.Meter(
		instrumentationName,
		otelmetric.WithInstrumentationVersion(Version()),
	)
	httpServerDuration, err := meter.Int64Histogram("http.server.duration", otelmetric.WithUnit("ms"))
	if err != nil {
		otel.Handle(err)
	}

	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	return func(c *gin.Context) {
		for _, f := range cfg.Filters {
			if !f(c.Request) {
				// Serve the request to the next middleware
				// if a filter rejects the request.
				c.Next()
				return
			}
		}
		c.Set(tracerKey, tracer)
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()
		ctx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(c.Request.Header))
		spanAttrs := semconvutil.HTTPServerRequest(service, c.Request)
		var spanName string
		if cfg.SpanNameFormatter == nil {
			spanName = c.FullPath()
		} else {
			spanName = cfg.SpanNameFormatter(c.Request)
		}
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		} else {
			rAttr := semconv.HTTPRoute(spanName)
			spanAttrs = append(spanAttrs, rAttr)
		}
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(spanAttrs...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}
		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)

		var httpStatus int
		defer func(t time.Time) {
			elapsedTime := time.Since(t) / time.Millisecond
			var metricAttrs []attribute.KeyValue
			// Filter span attributes to the ones allowed for server metrics.
			for _, attr := range spanAttrs {
				switch attr.Key {
				case semconv.HTTPMethodKey, semconv.HTTPSchemeKey, semconv.HTTPRouteKey,
					semconv.HTTPFlavorKey, semconv.NetHostNameKey, semconv.NetHostPortKey:
					metricAttrs = append(metricAttrs, attr)
				}
			}
			if httpStatus > 0 {
				metricAttrs = append(metricAttrs, semconv.HTTPStatusCodeKey.Int(httpStatus))
			}
			o := otelmetric.WithAttributes(metricAttrs...)
			httpServerDuration.Record(ctx, int64(elapsedTime), o)
		}(time.Now())

		// serve the request to the next middleware
		c.Next()

		httpStatus = c.Writer.Status()
		span.SetStatus(semconvutil.HTTPServerStatus(httpStatus))
		if httpStatus > 0 {
			span.SetAttributes(semconv.HTTPStatusCode(httpStatus))
		}
		if len(c.Errors) > 0 {
			span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
		}
	}
}

// HTML will trace the rendering of the template as a child of the
// span in the given context. This is a replacement for
// gin.Context.HTML function - it invokes the original function after
// setting up the span.
func HTML(c *gin.Context, code int, name string, obj interface{}) {
	var tracer oteltrace.Tracer
	tracerInterface, ok := c.Get(tracerKey)
	if ok {
		tracer, ok = tracerInterface.(oteltrace.Tracer)
	}
	if !ok {
		tracer = otel.GetTracerProvider().Tracer(
			instrumentationName,
			oteltrace.WithInstrumentationVersion(Version()),
		)
	}
	savedContext := c.Request.Context()
	defer func() {
		c.Request = c.Request.WithContext(savedContext)
	}()
	opt := oteltrace.WithAttributes(attribute.String("go.template", name))
	_, span := tracer.Start(savedContext, "gin.renderer.html", opt)
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
	c.HTML(code, name, obj)
}

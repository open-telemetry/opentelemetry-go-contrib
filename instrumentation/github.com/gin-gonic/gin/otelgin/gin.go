// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace.go

package otelgin // import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin/internal/semconv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerKey = "otel-go-contrib-tracer"
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		oteltrace.WithInstrumentationVersion(Version()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	if cfg.SpanNameFormatter == nil {
		cfg.SpanNameFormatter = defaultSpanNameFormatter
	}

	meter := cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
	)

	sc := semconv.NewHTTPServer(meter)

	return func(c *gin.Context) {
		requestStartTime := time.Now()

		for _, f := range cfg.Filters {
			if !f(c.Request) {
				// Serve the request to the next middleware
				// if a filter rejects the request.
				c.Next()
				return
			}
		}
		for _, f := range cfg.GinFilters {
			if !f(c) {
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

		requestTraceAttrOpts := semconv.RequestTraceAttrsOpts{
			// Gin's ClientIP method can detect the client's IP from various headers set by proxies, and it's configurable
			HTTPClientIP: c.ClientIP(),
		}

		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(sc.RequestTraceAttrs(service, c.Request, requestTraceAttrOpts)...),
			oteltrace.WithAttributes(sc.Route(c.FullPath())),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}

		opts = append(opts, cfg.SpanStartOptions...)

		spanName := cfg.SpanNameFormatter(c)
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		}
		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)

		// serve the request to the next middleware
		c.Next()

		status := c.Writer.Status()
		span.SetStatus(sc.Status(status))
		if status > 0 {
			span.SetAttributes(semconv.HTTPStatusCode(status))
		}
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}

		// Record the server-side attributes.
		var additionalAttributes []attribute.KeyValue
		if c.FullPath() != "" {
			additionalAttributes = append(additionalAttributes, sc.Route(c.FullPath()))
		}
		if cfg.MetricAttributeFn != nil {
			additionalAttributes = append(additionalAttributes, cfg.MetricAttributeFn(c.Request)...)
		}
		if cfg.GinMetricAttributeFn != nil {
			additionalAttributes = append(additionalAttributes, cfg.GinMetricAttributeFn(c)...)
		}

		sc.RecordMetrics(ctx, semconv.ServerMetricData{
			ServerName:   service,
			ResponseSize: int64(c.Writer.Size()),
			MetricAttributes: semconv.MetricAttributes{
				Req:                  c.Request,
				StatusCode:           status,
				AdditionalAttributes: additionalAttributes,
			},
			MetricData: semconv.MetricData{
				RequestSize: c.Request.ContentLength,
				ElapsedTime: float64(time.Since(requestStartTime)) / float64(time.Millisecond),
			},
		})
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
			ScopeName,
			oteltrace.WithInstrumentationVersion(Version()),
		)
	}
	savedContext := c.Request.Context()
	defer func() {
		c.Request = c.Request.WithContext(savedContext)
	}()
	opt := oteltrace.WithAttributes(attribute.String("go.template", name))
	_, span := tracer.Start(savedContext, "gin.renderer.html", opt)
	defer span.End()
	c.HTML(code, name, obj)
}

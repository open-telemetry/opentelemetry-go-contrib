// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho/internal/semconv"
)

const (
	tracerKey = "otel-go-contrib-tracer-labstack-echo"
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// Middleware returns echo middleware which will trace incoming requests.
func Middleware(serverName string, opts ...Option) echo.MiddlewareFunc {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		oteltrace.WithInstrumentationVersion(Version),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	if cfg.Skipper == nil {
		cfg.Skipper = middleware.DefaultSkipper
	}
	if cfg.OnError == nil {
		cfg.OnError = defaultOnError
	}

	meter := cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
	)

	semconvSrv := semconv.NewHTTPServer(meter)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestStartTime := time.Now()
			if cfg.Skipper(c) {
				return next(c)
			}

			c.Set(tracerKey, tracer)
			request := c.Request()
			savedCtx := request.Context()
			defer func() {
				request = request.WithContext(savedCtx)
				c.SetRequest(request)
			}()
			ctx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(request.Header))
			opts := []oteltrace.SpanStartOption{
				oteltrace.WithAttributes(
					semconvSrv.RequestTraceAttrs(serverName, request, semconv.RequestTraceAttrsOpts{})...,
				),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			}
			if path := c.Path(); path != "" {
				rAttr := semconvSrv.Route(path)
				opts = append(opts, oteltrace.WithAttributes(rAttr))
			}
			spanName := spanNameFormatter(c)

			ctx, span := tracer.Start(ctx, spanName, opts...)
			defer span.End()

			// pass the span through the request context
			c.SetRequest(request.WithContext(ctx))

			// serve the request to the next middleware
			err := next(c)
			if err != nil {
				span.SetAttributes(attribute.String("echo.error", err.Error()))
				cfg.OnError(c, err)
			}

			status := c.Response().Status
			span.SetStatus(semconvSrv.Status(status))
			span.SetAttributes(semconvSrv.ResponseTraceAttrs(semconv.ResponseTelemetry{
				StatusCode: status,
				WriteBytes: c.Response().Size,
			})...)

			// Record the server-side attributes.
			var additionalAttributes []attribute.KeyValue
			if path := c.Path(); path != "" {
				additionalAttributes = append(additionalAttributes, semconvSrv.Route(path))
			}
			if cfg.MetricAttributeFn != nil {
				additionalAttributes = append(additionalAttributes, cfg.MetricAttributeFn(request)...)
			}
			if cfg.EchoMetricAttributeFn != nil {
				additionalAttributes = append(additionalAttributes, cfg.EchoMetricAttributeFn(c)...)
			}

			semconvSrv.RecordMetrics(ctx, semconv.ServerMetricData{
				ServerName:   serverName,
				ResponseSize: c.Response().Size,
				MetricAttributes: semconv.MetricAttributes{
					Req:                  request,
					StatusCode:           status,
					AdditionalAttributes: additionalAttributes,
				},
				MetricData: semconv.MetricData{
					RequestSize:     request.ContentLength,
					RequestDuration: time.Since(requestStartTime),
				},
			})

			return err
		}
	}
}

func spanNameFormatter(c echo.Context) string {
	method, path := strings.ToUpper(c.Request().Method), c.Path()
	if !slices.Contains([]string{
		http.MethodGet, http.MethodHead,
		http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete,
		http.MethodConnect, http.MethodOptions,
		http.MethodTrace,
	}, method) {
		method = "HTTP"
	}

	if path != "" {
		return method + " " + path
	}

	return method
}

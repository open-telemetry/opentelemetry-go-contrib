// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelecho

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.43.0"
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

			// Call the error hook before capturing the response status: the
			// default OnError writes the error response via c.Error, which is
			// what makes c.Response().Status reflect the error status code.
			if err != nil {
				cfg.OnError(c, err)
			}

			status := c.Response().Status
			spanCode, spanMsg := semconvSrv.Status(status)
			span.SetStatus(spanCode, spanMsg)
			span.SetAttributes(semconvSrv.ResponseTraceAttrs(semconv.ResponseTelemetry{
				StatusCode: status,
				WriteBytes: c.Response().Size,
			})...)

			// Classify the failure cause with a low-cardinality error.type
			// instead of recording the full error string as an unbounded
			// "echo.error" attribute. A cancelled request context (client
			// disconnect or deadline) is always an error, independent of the
			// response status code; other handler errors are only classified
			// when they surface as a server error, since Echo handlers
			// idiomatically return *echo.HTTPError for 4xx/3xx responses.
			var errorTypeAttr attribute.KeyValue
			switch {
			case err != nil:
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || status >= 500 {
					span.SetStatus(codes.Error, err.Error())
					errorTypeAttr = errorTypeAttrFrom(err)
				} else if reqErr := c.Request().Context().Err(); reqErr != nil {
					// The handler returned an error that does not surface as a
					// server error (e.g. an *echo.HTTPError for a 4xx/3xx
					// response), but the request context may still be the actual
					// failure cause: a client disconnect or deadline observed
					// while the handler was running.
					span.SetStatus(codes.Error, reqErr.Error())
					errorTypeAttr = errorTypeAttrFrom(reqErr)
				}
			default:
				if reqErr := c.Request().Context().Err(); reqErr != nil {
					span.SetStatus(codes.Error, reqErr.Error())
					errorTypeAttr = errorTypeAttrFrom(reqErr)
				}
			}
			if errorTypeAttr.Valid() {
				span.SetAttributes(errorTypeAttr)
			}

			// Record the server-side attributes.
			var additionalAttributes []attribute.KeyValue
			if cfg.MetricAttributeFn != nil {
				additionalAttributes = append(additionalAttributes, cfg.MetricAttributeFn(request)...)
			}
			if cfg.EchoMetricAttributeFn != nil {
				additionalAttributes = append(additionalAttributes, cfg.EchoMetricAttributeFn(c)...)
			}
			if errorTypeAttr.Valid() {
				additionalAttributes = append(additionalAttributes, errorTypeAttr)
			}

			semconvSrv.RecordMetrics(ctx, semconv.ServerMetricData{
				ServerName:   serverName,
				ResponseSize: c.Response().Size,
				MetricAttributes: semconv.MetricAttributes{
					Req:                  request,
					StatusCode:           status,
					Route:                c.Path(),
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

// errorTypeAttrFrom returns a low-cardinality error.type attribute for err.
// Context cancellation and deadline errors are classified with their
// canonical values so a client disconnect is distinguishable from a genuine
// server fault; other errors fall back to their concrete type.
func errorTypeAttrFrom(err error) attribute.KeyValue {
	switch {
	case errors.Is(err, context.Canceled):
		return otelsemconv.ErrorTypeKey.String("context_canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return otelsemconv.ErrorTypeKey.String("context_deadline_exceeded")
	default:
		return otelsemconv.ErrorType(err)
	}
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/semconv"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/request"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// Middleware sets up a handler to start tracing the incoming
// requests.  The service parameter should describe the name of the
// (virtual) server handling the request.
func Middleware(service string, opts ...Option) mux.MiddlewareFunc {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.spanNameFormatter == nil {
		cfg.spanNameFormatter = defaultSpanNameFunc
	}

	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}

	cfg.Meter = cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
	)

	return func(handler http.Handler) http.Handler {
		return traceware{
			service:            service,
			tracer:             tracer,
			propagators:        cfg.Propagators,
			handler:            handler,
			spanNameFormatter:  cfg.spanNameFormatter,
			publicEndpoint:     cfg.PublicEndpoint,
			publicEndpointFn:   cfg.PublicEndpointFn,
			filters:            cfg.Filters,
			metricAttributesFn: cfg.MetricAttributesFn,
			semconv:            semconv.NewHTTPServer(cfg.Meter),
		}
	}
}

type traceware struct {
	service            string
	tracer             trace.Tracer
	propagators        propagation.TextMapPropagator
	handler            http.Handler
	spanNameFormatter  func(string, *http.Request) string
	publicEndpoint     bool
	publicEndpointFn   func(*http.Request) bool
	filters            []Filter
	metricAttributesFn func(*http.Request) []attribute.KeyValue
	semconv            semconv.HTTPServer
}

// defaultSpanNameFunc just reuses the route name as the span name.
func defaultSpanNameFunc(routeName string, _ *http.Request) string { return routeName }

// ServeHTTP implements the http.Handler interface. It does the actual
// tracing of the request.
func (tw traceware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestStartTime := time.Now()
	for _, f := range tw.filters {
		if !f(r) {
			// Simply pass through to the handler if a filter rejects the request
			tw.handler.ServeHTTP(w, r)
			return
		}
	}

	ctx := tw.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	opts := []trace.SpanStartOption{
		trace.WithAttributes(tw.semconv.RequestTraceAttrs(tw.service, r)...),
	}

	if tw.publicEndpoint || (tw.publicEndpointFn != nil && tw.publicEndpointFn(r.WithContext(ctx))) {
		opts = append(opts, trace.WithNewRoot())
		// Linking incoming span context if any for public endpoint.
		if s := trace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
			opts = append(opts, trace.WithLinks(trace.Link{SpanContext: s}))
		}
	}

	routeStr := fmt.Sprintf("HTTP %s route not found", r.Method)
	if route := mux.CurrentRoute(r); route != nil {
		if pathTemplate, err := route.GetPathTemplate(); err == nil {
			routeStr = pathTemplate
		} else if pathRegexp, err := route.GetPathRegexp(); err == nil {
			routeStr = pathRegexp
		}
	}

	if routeStr != fmt.Sprintf("HTTP %s route not found", r.Method) {
		rAttr := tw.semconv.Route(routeStr)
		opts = append(opts, trace.WithAttributes(rAttr))
	}

	ctx, span := tw.tracer.Start(ctx, tw.spanNameFormatter(routeStr, r), opts...)
	defer span.End()

	var readRecordFunc, writeRecordFunc func(int64)

	// if request body is nil or NoBody, we don't want to mutate the body as it
	// will affect the identity of it in an unforeseeable way because we assert
	// ReadCloser fulfills a certain interface and it is indeed nil or NoBody.
	bw := request.NewBodyWrapper(r.Body, readRecordFunc)
	if r.Body != nil && r.Body != http.NoBody {
		r.Body = bw
	}

	rrw := request.NewRespWriterWrapper(w, writeRecordFunc)

	tw.handler.ServeHTTP(w, r.WithContext(ctx))
	bytesWritten := rrw.BytesWritten()
	statusCode := rrw.StatusCode()

	if statusCode > 0 {
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))
	}

	errCode, errDesc := tw.semconv.Status(statusCode)
	span.SetStatus(errCode, errDesc)
	span.SetAttributes(tw.semconv.ResponseTraceAttrs(semconv.ResponseTelemetry{
		StatusCode: statusCode,
		ReadBytes:  bw.BytesRead(),
		ReadError:  bw.Error(),
		WriteBytes: bytesWritten,
		WriteError: rrw.Error(),
	})...)

	// Use floating point division here for higher precision (instead of Millisecond method).
	elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)

	var attrs []attribute.KeyValue
	metricAttributes := semconv.MetricAttributes{
		Req:                  r,
		StatusCode:           statusCode,
		AdditionalAttributes: append(attrs, tw.metricAttributesFromRequest(r)...),
	}

	tw.semconv.RecordMetrics(ctx, semconv.ServerMetricData{
		ServerName:       tw.service,
		ResponseSize:     bytesWritten,
		MetricAttributes: metricAttributes,
		MetricData: semconv.MetricData{
			RequestSize: bw.BytesRead(),
			ElapsedTime: elapsedTime,
		},
	})
}

func (tw traceware) metricAttributesFromRequest(r *http.Request) []attribute.KeyValue {
	var attributeForRequest []attribute.KeyValue
	if tw.metricAttributesFn != nil {
		attributeForRequest = tw.metricAttributesFn(r)
	}
	return attributeForRequest
}

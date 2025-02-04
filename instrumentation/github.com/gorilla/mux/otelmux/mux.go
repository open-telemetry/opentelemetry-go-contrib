// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

import (
	"fmt"
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/request"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/semconv"

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
	meter := cfg.MeterProvider.Meter(
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
			meter:              meter,
			semconv:            semconv.NewHTTPServer(meter),
			metricAttributesFn: cfg.MetricAttributesFn,
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
	meter              metric.Meter
	semconv            semconv.HTTPServer
	metricAttributesFn func(*http.Request) []attribute.KeyValue
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
		trace.WithSpanKind(trace.SpanKindServer),
	}

	if tw.publicEndpoint || (tw.publicEndpointFn != nil && tw.publicEndpointFn(r.WithContext(ctx))) {
		opts = append(opts, trace.WithNewRoot())
		// Linking incoming span context if any for public endpoint.
		if s := trace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
			opts = append(opts, trace.WithLinks(trace.Link{SpanContext: s}))
		}
	}

	routeStr := ""
	route := mux.CurrentRoute(r)
	if route != nil {
		routeStr, _ = route.GetPathTemplate()
		if routeStr == "" {
			routeStr, _ = route.GetPathRegexp()
		}
	}

	if routeStr == "" {
		routeStr = fmt.Sprintf("HTTP %s route not found", r.Method)
	} else {
		rAttr := tw.semconv.Route(routeStr)
		opts = append(opts, trace.WithAttributes(rAttr))
	}
	ctx, span := tw.tracer.Start(ctx, tw.spanNameFormatter(routeStr, r), opts...)
	defer span.End()

	readRecordFunc := func(int64) {}
	// if request body is nil or NoBody, we don't want to mutate the body as it
	// will affect the identity of it in an unforeseeable way because we assert
	// ReadCloser fulfills a certain interface and it is indeed nil or NoBody.
	bw := request.NewBodyWrapper(r.Body, readRecordFunc)
	if r.Body != nil && r.Body != http.NoBody {
		r.Body = bw
	}

	writeRecordFunc := func(int64) {}
	rww := request.NewRespWriterWrapper(w, writeRecordFunc)

	// Wrap w to use our ResponseWriter methods while also exposing
	// other interfaces that w may implement (http.CloseNotifier,
	// http.Flusher, http.Hijacker, http.Pusher, io.ReaderFrom).
	w = httpsnoop.Wrap(w, httpsnoop.Hooks{
		Header: func(httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
			return rww.Header
		},
		Write: func(httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return rww.Write
		},
		WriteHeader: func(httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return rww.WriteHeader
		},
		Flush: func(httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return rww.Flush
		},
	})

	tw.handler.ServeHTTP(w, r.WithContext(ctx))
	statusCode := rww.StatusCode()
	span.SetStatus(tw.semconv.Status(statusCode))
	span.SetAttributes(tw.semconv.ResponseTraceAttrs(semconv.ResponseTelemetry{
		StatusCode: statusCode,
		ReadBytes:  bw.BytesRead(),
		ReadError:  bw.Error(),
		WriteBytes: rww.BytesWritten(),
		WriteError: rww.Error(),
	})...)

	// Use floating point division here for higher precision (instead of Millisecond method).
	elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)

	metricAttributes := semconv.MetricAttributes{
		Req:                  r,
		StatusCode:           statusCode,
		AdditionalAttributes: tw.metricAttributesFromRequest(r),
	}

	tw.semconv.RecordMetrics(ctx, semconv.ServerMetricData{
		ServerName:       tw.service,
		ResponseSize:     rww.BytesWritten(),
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

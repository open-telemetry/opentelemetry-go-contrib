// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

import (
	"net/http"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/semconvutil"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/gorilla/mux"

	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
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
	// TODO: check if this is necessary
	_ = cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.spanNameFormatter == nil {
		cfg.spanNameFormatter = defaultSpanNameFunc
	}
	if cfg.meterProvider == nil {
		cfg.meterProvider = otel.GetMeterProvider()
	}

	return func(handler http.Handler) http.Handler {
		return middleware{
			service:           service,
			handler:           handler,
			spanNameFormatter: cfg.spanNameFormatter,
			tracerProvider:    cfg.TracerProvider,
			meterProvider:     cfg.meterProvider,
			propagators:       cfg.Propagators,
			publicEndpointFn:  cfg.PublicEndpointFn,
			publicEndpoint:    cfg.PublicEndpoint,
			filters:           cfg.Filters,
		}
	}
}

type middleware struct {
	handler           http.Handler
	service           string
	spanNameFormatter func(string, *http.Request) string
	tracerProvider    trace.TracerProvider
	propagators       propagation.TextMapPropagator
	publicEndpointFn  func(*http.Request) bool
	publicEndpoint    bool
	filters           []Filter
	meterProvider     metric.MeterProvider
}

func (m middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	routeStr := ""
	route := mux.CurrentRoute(request)
	if route != nil {
		var err error
		routeStr, err = route.GetPathTemplate()
		if err != nil {
			routeStr, err = route.GetPathRegexp()
			if err != nil {
				routeStr = ""
			}
		}
	}

	otelOpts := []otelhttp.Option{
		otelhttp.WithTracerProvider(m.tracerProvider),
		otelhttp.WithMeterProvider(m.meterProvider),
		otelhttp.WithPropagators(m.propagators),
		otelhttp.WithPublicEndpointFn(m.publicEndpointFn),
		otelhttp.WithSpanNameFormatter(func(op string, r *http.Request) string {
			return m.spanNameFormatter(routeStr, r)
		}),
	}

	if m.publicEndpoint {
		otelOpts = append(otelOpts, otelhttp.WithPublicEndpoint())
	}

	for _, f := range m.filters {
		otelOpts = append(otelOpts, otelhttp.WithFilter(otelhttp.Filter(f)))
	}
	traceOpts := []trace.SpanStartOption{
		trace.WithAttributes(semconvutil.HTTPServerRequest(m.service, request)...),
		trace.WithSpanKind(trace.SpanKindServer),
	}

	if routeStr != "" {
		rAttr := semconv.HTTPRoute(routeStr)
		traceOpts = append(traceOpts, trace.WithAttributes(rAttr))
	}
	otelOpts = append(otelOpts, otelhttp.WithSpanOptions(traceOpts...))

	h := otelhttp.NewHandler(
		m.handler,
		m.service,
		otelOpts...,
	)
	h.ServeHTTP(writer, request)
}

var _ http.Handler = middleware{}

// defaultSpanNameFunc just reuses the route name as the span name.
func defaultSpanNameFunc(routeName string, r *http.Request) string {
	return routeName
}

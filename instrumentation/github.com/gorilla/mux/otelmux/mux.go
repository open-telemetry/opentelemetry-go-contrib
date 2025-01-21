// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

import (
	"net/http"

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
	_ = cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.spanNameFormatter == nil {
		cfg.spanNameFormatter = defaultSpanNameFunc
	} else {
		tmp := cfg.spanNameFormatter
		cfg.spanNameFormatter = func(op string, r *http.Request) string {
			routeStr := ""
			route := mux.CurrentRoute(r)
			if route != nil {
				var err error
				routeStr, err = route.GetPathTemplate()
				if err != nil {
					routeStr, err = route.GetPathRegexp()
					if err != nil {
						routeStr = op
					}
				}
			}
			return tmp(routeStr, r)
		}
	}

	otelOpts := []otelhttp.Option{
		otelhttp.WithTracerProvider(cfg.TracerProvider),
		otelhttp.WithPropagators(cfg.Propagators),
		otelhttp.WithSpanNameFormatter(cfg.spanNameFormatter),
		otelhttp.WithPublicEndpointFn(cfg.PublicEndpointFn),
		otelhttp.WithSpanOptions(),
	}
	if cfg.PublicEndpoint {
		otelOpts = append(otelOpts, otelhttp.WithPublicEndpoint())
	}

	for _, f := range cfg.Filters {
		otelOpts = append(otelOpts, otelhttp.WithFilter(otelhttp.Filter(f)))
	}

	return func(handler http.Handler) http.Handler {
		return middleware{
			service:  service,
			handler:  handler,
			otelOpts: otelOpts,
		}
	}
}

type middleware struct {
	handler  http.Handler
	service  string
	otelOpts []otelhttp.Option
}

func (m middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	traceOpts := []trace.SpanStartOption{
		trace.WithAttributes(semconvutil.HTTPServerRequest(m.service, request)...),
		trace.WithSpanKind(trace.SpanKindServer),
	}

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

	if routeStr != "" {
		rAttr := semconv.HTTPRoute(routeStr)
		traceOpts = append(traceOpts, trace.WithAttributes(rAttr))
	}
	m.otelOpts = append(m.otelOpts, otelhttp.WithSpanOptions(traceOpts...))

	h := otelhttp.NewHandler(
		m.handler,
		m.service,
		m.otelOpts...,
	)
	h.ServeHTTP(writer, request)
}

var _ http.Handler = middleware{}

// defaultSpanNameFunc just reuses the route name as the span name.
func defaultSpanNameFunc(routeName string, r *http.Request) string {
	routeStr := routeName
	route := mux.CurrentRoute(r)
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
	return routeStr
}

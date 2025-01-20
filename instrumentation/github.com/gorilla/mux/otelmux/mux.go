// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

import (
	"fmt"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/semconvutil"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/request"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

	return func(handler http.Handler) http.Handler {
		return traceware{
			service:           service,
			tracer:            tracer,
			propagators:       cfg.Propagators,
			handler:           handler,
			spanNameFormatter: cfg.spanNameFormatter,
			publicEndpoint:    cfg.PublicEndpoint,
			publicEndpointFn:  cfg.PublicEndpointFn,
			filters:           cfg.Filters,
		}
	}
}

type traceware struct {
	service           string
	tracer            trace.Tracer
	propagators       propagation.TextMapPropagator
	handler           http.Handler
	spanNameFormatter func(string, *http.Request) string
	publicEndpoint    bool
	publicEndpointFn  func(*http.Request) bool
	filters           []Filter
}

// defaultSpanNameFunc just reuses the route name as the span name.
func defaultSpanNameFunc(routeName string, _ *http.Request) string { return routeName }

// ServeHTTP implements the http.Handler interface. It does the actual
// tracing of the request.
func (tw traceware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, f := range tw.filters {
		if !f(r) {
			// Simply pass through to the handler if a filter rejects the request
			tw.handler.ServeHTTP(w, r)
			return
		}
	}

	ctx := tw.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	routeStr := ""
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

	opts := []trace.SpanStartOption{
		trace.WithAttributes(semconvutil.HTTPServerRequest(tw.service, r)...),
		trace.WithSpanKind(trace.SpanKindServer),
	}

	if tw.publicEndpoint || (tw.publicEndpointFn != nil && tw.publicEndpointFn(r.WithContext(ctx))) {
		opts = append(opts, trace.WithNewRoot())
		// Linking incoming span context if any for public endpoint.
		if s := trace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
			opts = append(opts, trace.WithLinks(trace.Link{SpanContext: s}))
		}
	}

	if routeStr == "" {
		routeStr = fmt.Sprintf("HTTP %s route not found", r.Method)
	} else {
		rAttr := semconv.HTTPRoute(routeStr)
		opts = append(opts, trace.WithAttributes(rAttr))
	}
	spanName := tw.spanNameFormatter(routeStr, r)
	ctx, span := tw.tracer.Start(ctx, spanName, opts...)
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
	if statusCode > 0 {
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))
	}
	span.SetStatus(semconvutil.HTTPServerStatus(statusCode))
}

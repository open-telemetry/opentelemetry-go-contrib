// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/semconv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
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
			semconv:           semconv.NewHTTPServer(noop.Meter{}),
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
	semconv           semconv.HTTPServer
}

type recordingResponseWriter struct {
	writer  http.ResponseWriter
	written bool
	status  int

	bytesWritten int64 // Track bytes written to the response
	writeError   error // Track errors from writing to the response
}

var rrwPool = &sync.Pool{
	New: func() interface{} {
		return &recordingResponseWriter{}
	},
}

func getRRW(writer http.ResponseWriter) *recordingResponseWriter {
	rrw := rrwPool.Get().(*recordingResponseWriter)
	rrw.written = false
	rrw.status = http.StatusOK
	rrw.writer = httpsnoop.Wrap(writer, httpsnoop.Hooks{
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				if !rrw.written {
					rrw.written = true
				}
				n, err := next(b)
				rrw.bytesWritten += int64(n)
				if err != nil {
					rrw.writeError = err
				}
				return n, err
			}
		},
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(statusCode int) {
				if !rrw.written {
					rrw.written = true
					rrw.status = statusCode
				}

				next(statusCode)
			}
		},
	})
	return rrw
}

func putRRW(rrw *recordingResponseWriter) {
	rrw.writer = nil
	rrw.bytesWritten = 0
	rrw.writeError = nil
	rrwPool.Put(rrw)
}

type bodyWrapper struct {
	io.ReadCloser
	OnRead func(n int64) // must not be nil

	mu   sync.Mutex
	read int64
	err  error
}

func newBodyWrapper(body io.ReadCloser, onRead func(int64)) *bodyWrapper {
	return &bodyWrapper{
		ReadCloser: body,
		OnRead:     onRead,
	}
}

// Read reads the data from the io.ReadCloser, and stores the number of bytes
// read and the error.
func (w *bodyWrapper) Read(b []byte) (int, error) {
	n, err := w.ReadCloser.Read(b)
	n1 := int64(n)

	w.updateReadData(n1, err)
	w.OnRead(n1)
	return n, err
}

func (w *bodyWrapper) updateReadData(n int64, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.read += n
	if err != nil {
		w.err = err
	}
}

// Close closes the io.ReadCloser.
func (w *bodyWrapper) Close() error {
	return w.ReadCloser.Close()
}

// BytesRead returns the number of bytes read up to this point.
func (w *bodyWrapper) BytesRead() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.read
}

// Error returns the last error.
func (w *bodyWrapper) Error() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.err
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

	// if request body is nil or NoBody, we don't want to mutate the body as it
	// will affect the identity of it in an unforeseeable way because we assert
	// ReadCloser fulfills a certain interface and it is indeed nil or NoBody.
	bw := newBodyWrapper(r.Body, func(int64) {})
	if r.Body != nil && r.Body != http.NoBody {
		r.Body = bw
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
	r2 := r.WithContext(ctx)
	rrw := getRRW(w)
	defer putRRW(rrw)
	tw.handler.ServeHTTP(rrw.writer, r2)
	if rrw.status > 0 {
		span.SetAttributes(semconv.HTTPStatusCode(rrw.status))
	}
	span.SetStatus(tw.semconv.Status(rrw.status))

	span.SetAttributes(tw.semconv.ResponseTraceAttrs(semconv.ResponseTelemetry{
		StatusCode: rrw.status,
		ReadBytes:  bw.BytesRead(),
		ReadError:  bw.Error(),
		WriteBytes: rrw.bytesWritten,
		WriteError: rrw.writeError,
	})...)
}

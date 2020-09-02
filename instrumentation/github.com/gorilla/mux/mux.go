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

package mux

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	otelcontrib "go.opentelemetry.io/contrib"

	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/semconv"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux"
)

// Middleware sets up a handler to start tracing the incoming
// requests.  The service parameter should describe the name of the
// (virtual) server handling the request.
func Middleware(service string, opts ...Option) mux.MiddlewareFunc {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otelglobal.TraceProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otelglobal.Propagators()
	}
	return func(handler http.Handler) http.Handler {
		return traceware{
			service:     service,
			tracer:      tracer,
			propagators: cfg.Propagators,
			handler:     handler,
		}
	}
}

type traceware struct {
	service     string
	tracer      oteltrace.Tracer
	propagators otelpropagation.Propagators
	handler     http.Handler
}

type recordingResponseWriter struct {
	writer  http.ResponseWriter
	written bool
	status  int
}

func (w *recordingResponseWriter) Header() http.Header {
	return w.writer.Header()
}

func (w *recordingResponseWriter) Write(slice []byte) (int, error) {
	w.writeHeader(http.StatusOK)
	return w.writer.Write(slice)
}

func (w *recordingResponseWriter) WriteHeader(statusCode int) {
	w.writeHeader(statusCode)
	w.writer.WriteHeader(statusCode)
}

func (w *recordingResponseWriter) writeHeader(statusCode int) {
	if !w.written {
		w.written = true
		w.status = statusCode
	}
}

var rrwPool = &sync.Pool{
	New: func() interface{} {
		return &recordingResponseWriter{}
	},
}

func getRRW(writer http.ResponseWriter) *recordingResponseWriter {
	rrw := rrwPool.Get().(*recordingResponseWriter)
	rrw.written = false
	rrw.status = 0
	rrw.writer = writer
	return rrw
}

func putRRW(rrw *recordingResponseWriter) {
	rrw.writer = nil
	rrwPool.Put(rrw)
}

// ServeHTTP implements the http.Handler interface. It does the actual
// tracing of the request.
func (tw traceware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := otelpropagation.ExtractHTTP(r.Context(), tw.propagators, r.Header)
	spanName := ""
	route := mux.CurrentRoute(r)
	if route != nil {
		var err error
		spanName, err = route.GetPathTemplate()
		if err != nil {
			spanName, err = route.GetPathRegexp()
			if err != nil {
				spanName = ""
			}
		}
	}
	routeStr := spanName
	if spanName == "" {
		spanName = fmt.Sprintf("HTTP %s route not found", r.Method)
	}
	opts := []oteltrace.StartOption{
		oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
		oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
		oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(tw.service, routeStr, r)...),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	}
	ctx, span := tw.tracer.Start(ctx, spanName, opts...)
	defer span.End()
	r2 := r.WithContext(ctx)
	rrw := getRRW(w)
	defer putRRW(rrw)
	tw.handler.ServeHTTP(rrw, r2)
	attrs := semconv.HTTPAttributesFromHTTPStatusCode(rrw.status)
	spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rrw.status)
	span.SetAttributes(attrs...)
	span.SetStatus(spanStatus, spanMessage)
}

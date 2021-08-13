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

package otelchi

import (
	"net/http"
	"strings"
	"sync"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	otelcontrib "go.opentelemetry.io/contrib"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "github.com/riandyrn/otelchi"
)

// Middleware sets up a handler to start tracing the incoming
// requests. The serverName parameter should describe the name of the
// (virtual) server handling the request.
func Middleware(serverName string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	return func(handler http.Handler) http.Handler {
		return traceware{
			serverName:  serverName,
			tracer:      tracer,
			propagators: cfg.Propagators,
			handler:     handler,
		}
	}
}

type traceware struct {
	serverName  string
	tracer      oteltrace.Tracer
	propagators propagation.TextMapPropagator
	handler     http.Handler
}

type recordingResponseWriter struct {
	writer  http.ResponseWriter
	written bool
	status  int
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
	rrw.writer = httpsnoop.Wrap(writer, httpsnoop.Hooks{
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				if !rrw.written {
					rrw.written = true
					rrw.status = http.StatusOK
				}
				return next(b)
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
	rrwPool.Put(rrw)
}

// ServeHTTP implements the http.Handler interface. It does the actual
// tracing of the request.
func (tw traceware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := tw.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tw.tracer.Start(ctx, "", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	defer span.End()

	r2 := r.WithContext(ctx)
	rrw := getRRW(w)
	defer putRRW(rrw)
	tw.handler.ServeHTTP(rrw.writer, r2)

	routeStr := strings.Join(chi.RouteContext(r2.Context()).RoutePatterns, "")
	attrs := semconv.HTTPAttributesFromHTTPStatusCode(rrw.status)
	attrs = append(attrs, semconv.NetAttributesFromHTTPRequest("tcp", r2)...)
	attrs = append(attrs, semconv.EndUserAttributesFromHTTPRequest(r2)...)
	attrs = append(attrs, semconv.HTTPServerAttributesFromHTTPRequest(tw.serverName, routeStr, r2)...)
	attrs = append(
		attrs,
		semconv.HTTPTargetKey.String(r2.URL.String()),
	)
	span.SetAttributes(attrs...)
	span.SetName(routeStr)

	spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rrw.status)
	span.SetStatus(spanStatus, spanMessage)
}

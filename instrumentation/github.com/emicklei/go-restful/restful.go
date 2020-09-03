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

package restful

import (
	"github.com/emicklei/go-restful/v3"

	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/semconv"
)

const (
	tracerName    = "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful"
	tracerVersion = "1.0"
)

// OTelFilter returns a restful.FilterFunction which will trace an incoming request.
//
// The service parameter should describe the name of the (virtual) server handling
// the request.  Options can be applied to configure the tracer and propagators
// used for this filter.
func OTelFilter(service string, opts ...Option) restful.FilterFunction {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otelglobal.TraceProvider()
	}
	tracer := cfg.TracerProvider.Tracer(tracerName, oteltrace.WithInstrumentationVersion(tracerVersion))
	if cfg.Propagators == nil {
		cfg.Propagators = otelglobal.Propagators()
	}
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		r := req.Request
		ctx := otelpropagation.ExtractHTTP(r.Context(), cfg.Propagators, r.Header)
		route := req.SelectedRoutePath()
		spanName := route

		opts := []oteltrace.StartOption{
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, route, r)...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}
		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		req.Request = req.Request.WithContext(ctx)

		chain.ProcessFilter(req, resp)

		attrs := semconv.HTTPAttributesFromHTTPStatusCode(resp.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(resp.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}
}

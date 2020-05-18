package internal

import (
	"net/http"

	"go.opentelemetry.io/contrib/internal/trace"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

func StartTrace(r *http.Request, routeName string, conf Config) (*http.Request, oteltrace.Span) {
	ctx := r.Context()
	ctx = otelpropagation.ExtractHTTP(ctx, conf.Propagators, r.Header)
	opts := []oteltrace.StartOption{
		oteltrace.WithAttributes(trace.NetAttributesFromHTTPRequest("tcp", r)...),
		oteltrace.WithAttributes(trace.EndUserAttributesFromHTTPRequest(r)...),
		oteltrace.WithAttributes(trace.HTTPServerAttributesFromHTTPRequest(conf.Service, routeName, r)...),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	}

	ctx, span := conf.Tracer.Start(ctx, routeName, opts...)
	r = r.WithContext(ctx)
	return r, span
}

func EndTrace(span oteltrace.Span, status int) {
	attrs := trace.HTTPAttributesFromHTTPStatusCode(status)
	spanStatus, spanMessage := trace.SpanStatusFromHTTPStatusCode(status)

	span.SetAttributes(attrs...)
	span.SetStatus(spanStatus, spanMessage)
	span.End()
}

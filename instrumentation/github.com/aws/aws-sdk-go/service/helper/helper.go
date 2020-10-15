package helper

import (
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

func AppendSpanAndTraceIDFromSpan(attrs []label.KeyValue, span trace.Span) []label.KeyValue {
	linkSpanAttr := []label.KeyValue{
		label.String("span.id", span.SpanContext().SpanID.String()),
		label.String("trace.id", span.SpanContext().TraceID.String()),
	}

	return append(linkSpanAttr, attrs...)
}

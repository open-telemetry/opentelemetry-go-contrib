// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eventbridge // import "go.opentelemetry.io/contrib/processors/eventbridge"

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

var (
	logRecordObservedTimeUnixNanoKey   = attribute.Key("log.record.observed_time_unix_nano")
	logRecordSeverityNumberKey         = attribute.Key("log.record.severity_number")
	logRecordBodyKey                   = attribute.Key("log.record.body")
	logRecordDroppedAttributesCountKey = attribute.Key("log.record.dropped_attributes_count")
)

// LogProcessor is a [sdklog.Processor] implementation that bridges log-based
// events onto the current span as span events.
type LogProcessor struct{}

var _ sdklog.Processor = (*LogProcessor)(nil)

// NewLogProcessor returns a new [LogProcessor].
func NewLogProcessor() *LogProcessor {
	return new(LogProcessor)
}

// Enabled reports whether the Processor will process.
func (LogProcessor) Enabled(ctx context.Context, _ sdklog.EnabledParameters) bool {
	return trace.SpanFromContext(ctx).IsRecording()
}

// OnEmit bridges event records onto the current span when the record and span
// contexts match.
func (LogProcessor) OnEmit(ctx context.Context, record *sdklog.Record) error {
	if record == nil || record.EventName() == "" {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return nil
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return nil
	}

	if record.TraceID() != spanCtx.TraceID() || record.SpanID() != spanCtx.SpanID() {
		return nil
	}

	var opts []trace.EventOption
	if ts := record.Timestamp(); !ts.IsZero() {
		opts = append(opts, trace.WithTimestamp(ts))
	}

	if attrs := spanEventAttributes(record); len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}

	span.AddEvent(record.EventName(), opts...)
	return nil
}

// Shutdown is called when the [sdklog.Processor] is shutting down and is a
// no-op for this processor.
func (LogProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush is called to ensure all logs are flushed to the output and is a
// no-op for this processor.
func (LogProcessor) ForceFlush(context.Context) error { return nil }

func spanEventAttributes(record *sdklog.Record) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, record.AttributesLen()+4)

	record.WalkAttributes(func(kv log.KeyValue) bool {
		attrs = append(attrs, spanAttribute(kv))
		return true
	})

	if observed := record.ObservedTimestamp(); !observed.IsZero() {
		attrs = append(attrs, logRecordObservedTimeUnixNanoKey.Int64(observed.UnixNano()))
	}

	if sev := record.Severity(); sev != log.SeverityUndefined {
		attrs = append(attrs, logRecordSeverityNumberKey.Int64(int64(sev)))
	}

	if body := record.Body(); !body.Empty() {
		attrs = append(attrs, logRecordBodyKey.String(marshalLogValue(body)))
	}

	if dropped := record.DroppedAttributes(); dropped > 0 {
		attrs = append(attrs, logRecordDroppedAttributesCountKey.Int64(int64(dropped)))
	}

	return attrs
}

func spanAttribute(kv log.KeyValue) attribute.KeyValue {
	return attribute.KeyValue{
		Key:   attribute.Key(kv.Key),
		Value: spanAttributeValue(kv.Value),
	}
}

func spanAttributeValue(v log.Value) attribute.Value {
	switch v.Kind() {
	case log.KindBool:
		return attribute.BoolValue(v.AsBool())
	case log.KindFloat64:
		return attribute.Float64Value(v.AsFloat64())
	case log.KindInt64:
		return attribute.Int64Value(v.AsInt64())
	case log.KindString:
		return attribute.StringValue(v.AsString())
	case log.KindSlice:
		if attr, ok := spanSliceValue(v.AsSlice()); ok {
			return attr
		}
	}

	return attribute.StringValue(marshalLogValue(v))
}

func spanSliceValue(values []log.Value) (attribute.Value, bool) {
	if len(values) == 0 {
		return attribute.Value{}, false
	}

	switch values[0].Kind() {
	case log.KindBool:
		out := make([]bool, len(values))
		for i, v := range values {
			if v.Kind() != log.KindBool {
				return attribute.Value{}, false
			}
			out[i] = v.AsBool()
		}
		return attribute.BoolSliceValue(out), true
	case log.KindFloat64:
		out := make([]float64, len(values))
		for i, v := range values {
			if v.Kind() != log.KindFloat64 {
				return attribute.Value{}, false
			}
			out[i] = v.AsFloat64()
		}
		return attribute.Float64SliceValue(out), true
	case log.KindInt64:
		out := make([]int64, len(values))
		for i, v := range values {
			if v.Kind() != log.KindInt64 {
				return attribute.Value{}, false
			}
			out[i] = v.AsInt64()
		}
		return attribute.Int64SliceValue(out), true
	case log.KindString:
		out := make([]string, len(values))
		for i, v := range values {
			if v.Kind() != log.KindString {
				return attribute.Value{}, false
			}
			out[i] = v.AsString()
		}
		return attribute.StringSliceValue(out), true
	default:
		return attribute.Value{}, false
	}
}

func marshalLogValue(v log.Value) string {
	data, err := json.Marshal(logValueToAny(v))
	if err != nil {
		return v.String()
	}
	return string(data)
}

func logValueToAny(v log.Value) any {
	switch v.Kind() {
	case log.KindBool:
		return v.AsBool()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindString:
		return v.AsString()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		values := v.AsSlice()
		out := make([]any, len(values))
		for i, item := range values {
			out[i] = logValueToAny(item)
		}
		return out
	case log.KindMap:
		items := v.AsMap()
		out := make(map[string]any, len(items))
		for _, item := range items {
			out[item.Key] = logValueToAny(item.Value)
		}
		return out
	default:
		return nil
	}
}

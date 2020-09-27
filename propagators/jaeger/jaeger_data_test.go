package jaeger_test

import (
	"fmt"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	traceIDStr   = "4bf92f3577b34da6a3ce929d0e0e4736"
	spanIDStr    = "00f067aa0ba902b7"
	jaegerHeader = "uber-trace-id"
)

var (
	traceID = trace.ID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	spanID  = trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
)

type extractTest struct {
	name     string
	headers  map[string]string
	expected trace.SpanContext
}

var extractHeaders = []extractTest{
	{
		"empty",
		map[string]string{},
		trace.SpanContext{},
	},
	{
		"sampling state not sample",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:0", traceIDStr, spanIDStr),
		},
		trace.SpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
	},
	{
		"sampling state sampled",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceIDStr, spanIDStr),
		},
		trace.SpanContext{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		"sampling state debug",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:3", traceIDStr, spanIDStr),
		},
		trace.SpanContext{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled | trace.FlagsDebug,
		},
	},
	{
		"sampling state debug but sampled bit didn't set, result in not sampled decision",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:2", traceIDStr, spanIDStr),
		},
		trace.SpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
	},
}

var invalidExtractHeaders = []extractTest{}

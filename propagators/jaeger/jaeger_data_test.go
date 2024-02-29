// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaeger_test

import (
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

const (
	traceID15Str = "3ce929d0e0e4736"
	traceID16Str = "a3ce929d0e0e4736"
	traceID32Str = "a1ce929d0e0e4736a3ce929d0e0e4736"
	spanIDStr    = "00f067aa0ba902b7"
	jaegerHeader = "uber-trace-id"
)

var (
	traceID15 = trace.TraceID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	traceID16 = trace.TraceID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	traceID32 = trace.TraceID{0xa1, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	spanID    = trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
)

type extractTest struct {
	name     string
	headers  map[string]string
	expected trace.SpanContextConfig
	debug    bool
}

var extractHeaders = []extractTest{
	{
		"empty",
		map[string]string{},
		trace.SpanContextConfig{},
		false,
	},
	{
		"sampling state not sample",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:0", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		false,
	},
	{
		"sampling state sampled",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		false,
	},
	{
		"sampling state debug",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:3", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		true,
	},
	{
		"sampling state debug but sampled bit didn't set, result in not sampled decision",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:2", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		false,
	},
	{
		"flag can be various length",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:00001", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		false,
	},
	{
		"flag can be hex numbers",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:ff", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		true,
	},
	{
		"left padding 60 bit trace ID",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID15Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID15,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		false,
	},
	{
		"left padding 64 bit trace ID",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID16Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID16,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		false,
	},
	{
		"128 bit trace ID",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		false,
	},
	{
		"ignore parent span id",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:whatever:1", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		false,
	},
}

var invalidExtractHeaders = []extractTest{
	{
		name: "trace ID length > 32",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID32Str+"0000", spanIDStr),
		},
	},
	{
		name: "span ID length is not 16 or 32",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID32Str, spanIDStr+"0000"),
		},
	},
	{
		name: "invalid trace ID",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", "zcd00v0000000000a3ce929d0e0e4736", spanIDStr),
		},
	},
	{
		name: "invalid span ID",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID32Str, "00f0wiredba902b7"),
		},
	},
	{
		name: "invalid flags",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:wired", traceID32Str, spanIDStr),
		},
	},
	{
		name: "invalid separator",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s-%s-0-1", traceID32Str, spanIDStr),
		},
	},
	{
		name: "missing jaeger header",
		headers: map[string]string{
			jaegerHeader + "not": fmt.Sprintf("%s:%s:0:1", traceID32Str, spanIDStr),
		},
	},
	{
		name: "empty header value",
		headers: map[string]string{
			jaegerHeader: "",
		},
	},
}

type injectTest struct {
	name        string
	scc         trace.SpanContextConfig
	wantHeaders map[string]string
	debug       bool
}

var injectHeaders = []injectTest{
	{
		name: "sampled",
		scc: trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		wantHeaders: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", traceID32Str, spanIDStr),
		},
	},
	{
		name: "debug",
		scc: trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		wantHeaders: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:3", traceID32Str, spanIDStr),
		},
		debug: true,
	},
	{
		name: "not sampled",
		scc: trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		wantHeaders: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:0", traceID32Str, spanIDStr),
		},
	},
}

var invalidInjectHeaders = []injectTest{
	{
		name: "empty",
		scc:  trace.SpanContextConfig{},
	},
	{
		name: "missing traceID",
		scc: trace.SpanContextConfig{
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name: "missing spanID",
		scc: trace.SpanContextConfig{
			TraceID:    traceID32,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name: "missing both traceID and spanID",
		scc: trace.SpanContextConfig{
			TraceFlags: trace.FlagsSampled,
		},
	},
}

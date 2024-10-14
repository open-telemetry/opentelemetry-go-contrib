// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ot_test

import (
	"strings"

	"go.opentelemetry.io/otel/trace"
)

const (
	traceID16Str   = "a3ce929d0e0e4736"
	traceID32Str   = "a1ce929d0e0e4736a3ce929d0e0e4736"
	spanIDStr      = "00f067aa0ba902b7"
	traceIDHeader  = "ot-tracer-traceid"
	spanIDHeader   = "ot-tracer-spanid"
	sampledHeader  = "ot-tracer-sampled"
	baggageKey     = "test"
	baggageValue   = "value123"
	baggageHeader  = "ot-baggage-test"
	baggageKey2    = "test2"
	baggageValue2  = "value456"
	baggageHeader2 = "ot-baggage-test2"
)

var (
	traceID16    = trace.TraceID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	traceID32    = trace.TraceID{0xa1, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	spanID       = trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
	emptyBaggage = map[string]string{}
	baggageSet   = map[string]string{
		baggageKey: baggageValue,
	}
	baggageSet2 = map[string]string{
		baggageKey:  baggageValue,
		baggageKey2: baggageValue2,
	}
)

type extractTest struct {
	name     string
	headers  map[string]string
	expected trace.SpanContextConfig
	baggage  map[string]string
}

var extractHeaders = []extractTest{
	{
		"empty",
		map[string]string{},
		trace.SpanContextConfig{},
		emptyBaggage,
	},
	{
		"sampling state not sample",
		map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "0",
		},
		trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		emptyBaggage,
	},
	{
		"sampling state sampled",
		map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
			baggageHeader: baggageValue,
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		baggageSet,
	},
	{
		"baggage multiple values",
		map[string]string{
			traceIDHeader:  traceID32Str,
			spanIDHeader:   spanIDStr,
			sampledHeader:  "0",
			baggageHeader:  baggageValue,
			baggageHeader2: baggageValue2,
		},
		trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		baggageSet2,
	},
	{
		"left padding 64 bit trace ID",
		map[string]string{
			traceIDHeader: traceID16Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
		},
		trace.SpanContextConfig{
			TraceID:    traceID16,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		emptyBaggage,
	},
	{
		"128 bit trace ID",
		map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		emptyBaggage,
	},
}

var invalidExtractHeaders = []extractTest{
	{
		name: "trace ID length > 32",
		headers: map[string]string{
			traceIDHeader: traceID32Str + "0000",
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
		},
	},
	{
		name: "trace ID length is not 32 or 16",
		headers: map[string]string{
			traceIDHeader: "1234567890abcd01234",
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
		},
	},
	{
		name: "span ID length is not 16 or 32",
		headers: map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr + "0000",
			sampledHeader: "1",
		},
	},
	{
		name: "invalid trace ID",
		headers: map[string]string{
			traceIDHeader: "zcd00v0000000000a3ce929d0e0e4736",
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
		},
	},
	{
		name: "invalid span ID",
		headers: map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  "00f0wiredba902b7",
			sampledHeader: "1",
		},
	},
	{
		name: "invalid sampled",
		headers: map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "wired",
		},
	},
	{
		name: "invalid baggage key",
		headers: map[string]string{
			traceIDHeader:     traceID32Str,
			spanIDHeader:      spanIDStr,
			sampledHeader:     "1",
			"ot-baggage-d–76": "test",
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name: "invalid baggage value",
		headers: map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
			baggageHeader: "øtel",
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name: "invalid baggage result (too large)",
		headers: map[string]string{
			traceIDHeader: traceID32Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "1",
			baggageHeader: strings.Repeat("s", 8188),
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name:    "missing headers",
		headers: map[string]string{},
	},
	{
		name: "empty header value",
		headers: map[string]string{
			traceIDHeader: "",
		},
	},
}

type injectTest struct {
	name        string
	sc          trace.SpanContextConfig
	wantHeaders map[string]string
	baggage     map[string]string
}

var injectHeaders = []injectTest{
	{
		name: "sampled",
		sc: trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		wantHeaders: map[string]string{
			traceIDHeader: traceID16Str,
			spanIDHeader:  spanIDStr,
			sampledHeader: "true",
		},
	},
	{
		name: "not sampled",
		sc: trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		baggage: map[string]string{
			baggageKey:  baggageValue,
			baggageKey2: baggageValue2,
		},
		wantHeaders: map[string]string{
			traceIDHeader:  traceID16Str,
			spanIDHeader:   spanIDStr,
			sampledHeader:  "false",
			baggageHeader:  baggageValue,
			baggageHeader2: baggageValue2,
		},
	},
}

var invalidInjectHeaders = []injectTest{
	{
		name: "empty",
		sc:   trace.SpanContextConfig{},
	},
	{
		name: "missing traceID",
		sc: trace.SpanContextConfig{
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name: "missing spanID",
		sc: trace.SpanContextConfig{
			TraceID:    traceID32,
			TraceFlags: trace.FlagsSampled,
		},
	},
	{
		name: "missing both traceID and spanID",
		sc: trace.SpanContextConfig{
			TraceFlags: trace.FlagsSampled,
		},
	},
}

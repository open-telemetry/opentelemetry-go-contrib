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

package ot_test

import (
	"go.opentelemetry.io/otel/attribute"
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
	emptyBaggage = &attribute.Set{}
	// TODO: once baggage extraction is supported, re-enable this
	// baggageSet   = attribute.NewSet(
	// 	attribute.String(baggageKey, baggageValue),
	// 	attribute.String(baggageKey2, baggageValue2),
	// )
)

type extractTest struct {
	name     string
	headers  map[string]string
	expected trace.SpanContextConfig
	baggage  *attribute.Set
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
		emptyBaggage,
		// TODO: once baggage extraction is supported, re-enable this
		// &baggageSet,
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
	baggage     []attribute.KeyValue
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
			sampledHeader: "1",
		},
	},
	{
		name: "not sampled",
		sc: trace.SpanContextConfig{
			TraceID: traceID32,
			SpanID:  spanID,
		},
		baggage: []attribute.KeyValue{
			attribute.String(baggageKey, baggageValue),
			attribute.String(baggageKey2, baggageValue2),
		},
		wantHeaders: map[string]string{
			traceIDHeader:  traceID16Str,
			spanIDHeader:   spanIDStr,
			sampledHeader:  "0",
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

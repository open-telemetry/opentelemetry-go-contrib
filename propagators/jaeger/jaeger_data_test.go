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

package jaeger_test

import (
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

const (
	traceID16Str = "a3ce929d0e0e4736"
	traceID32Str = "a1ce929d0e0e4736a3ce929d0e0e4736"
	spanIDStr    = "00f067aa0ba902b7"
	jaegerHeader = "uber-trace-id"
)

var (
	traceID16 = trace.TraceID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	traceID32 = trace.TraceID{0xa1, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	spanID    = trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
)

type extractTest struct {
	name     string
	headers  map[string]string
	expected trace.SpanContextConfig
}

var extractHeaders = []extractTest{
	{
		"empty",
		map[string]string{},
		trace.SpanContextConfig{},
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
	},
	{
		"sampling state debug",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:3", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled | trace.FlagsDebug,
		},
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
	},
	{
		"flag can be hex numbers",
		map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:ff", traceID32Str, spanIDStr),
		},
		trace.SpanContextConfig{
			TraceID:    traceID32,
			SpanID:     spanID,
			TraceFlags: trace.FlagsDebug | trace.FlagsSampled,
		},
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
		name: "trace ID length is not 32 or 16",
		headers: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:1", "1234567890abcd01234", spanIDStr),
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
			TraceFlags: trace.FlagsSampled | trace.FlagsDebug,
		},
		wantHeaders: map[string]string{
			jaegerHeader: fmt.Sprintf("%s:%s:0:3", traceID32Str, spanIDStr),
		},
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

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

package spanname // import "go.opentelemetry.io/contrib/samplers/spanname"

import (
	"strings"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type ignoreSpansWithNameSampler struct {
	targets []string
}

// IgnoreSpansWithNameSampler drops all spans that contains substring in names
// in its span name.
//
// For example, if you have spans with names "sample.xxxxx", then IgnoreSpansWithSampler("sample")
// drops all the those spans.
func IgnoreSpansWithNameSampler(names ...string) sdktrace.Sampler {
	return ignoreSpansWithNameSampler{
		targets: names,
	}
}

func (s ignoreSpansWithNameSampler) Description() string {
	return "drop all spans with the name that contains one of targets names."
}

func (s ignoreSpansWithNameSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	ts := trace.SpanContextFromContext(p.ParentContext).TraceState()
	for _, t := range s.targets {
		if strings.Contains(p.Name, t) {
			return sdktrace.SamplingResult{
				Decision:   sdktrace.Drop,
				Tracestate: ts,
			}
		}
	}
	return sdktrace.SamplingResult{
		Decision:   sdktrace.RecordAndSample,
		Tracestate: ts,
	}
}

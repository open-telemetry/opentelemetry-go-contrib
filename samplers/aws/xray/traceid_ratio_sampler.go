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

package xray

import (
	"encoding/binary"
	"fmt"
	"math"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type traceIDRatioBasedSampler struct {
	sdktrace.Sampler
	max  uint64
	desc string
}

// NewTraceIDRatioBased creates a sampler based on random number.
// fraction parameter should be between 0 and 1 where:
// fraction >= 1 it will always sample
// fraction <= 0 it will never sample
func NewTraceIDRatioBased(fraction float64) sdktrace.Sampler {
	if fraction >= 1 {
		return sdktrace.AlwaysSample()
	} else if fraction <= 0 {
		return sdktrace.NeverSample()
	}

	return &traceIDRatioBasedSampler{
		desc: fmt.Sprintf("xrayTraceIDRatioBasedSampler{%v}", fraction),
		max:  uint64(fraction * math.MaxUint64),
	}
}

func (s *traceIDRatioBasedSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// The default otel sampler pick the first 8 bytes to make the sampling decision and this is a problem to
	// xray case as the first 4 bytes on the xray traceId is the time of the original request and the random part are
	// the 12 last bytes, and so, this sampler pick the last 8 bytes to make the sampling decision.
	// Xray Trace format: https://docs.aws.amazon.com/xray/latest/devguide/xray-api-sendingdata.html
	// Xray Id Generator: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/54f0bc5c0fd347cd6db9b7bc14c9f0c00dfcb36b/propagators/aws/xray/idgenerator.go#L58-L63
	// Ref: https://github.com/open-telemetry/opentelemetry-go/blob/7a60bc785d669fa6ad26ba70e88151d4df631d90/sdk/trace/sampling.go#L82-L95
	val := binary.BigEndian.Uint64(p.TraceID[8:16])
	psc := trace.SpanContextFromContext(p.ParentContext)
	shouldSample := val < s.max
	if shouldSample {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.RecordAndSample,
			Tracestate: psc.TraceState(),
		}
	}
	return sdktrace.SamplingResult{
		Decision:   sdktrace.Drop,
		Tracestate: psc.TraceState(),
	}
}

func (s *traceIDRatioBasedSampler) Description() string {
	return s.desc
}

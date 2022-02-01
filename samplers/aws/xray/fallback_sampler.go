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
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type FallbackSampler struct {
	currentEpoch   int64
	clock          Clock
	defaultSampler sdktrace.Sampler
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*FallbackSampler)(nil)

func NewFallbackSampler() *FallbackSampler {
	return &FallbackSampler{
		clock:          &DefaultClock{},
		defaultSampler: sdktrace.TraceIDRatioBased(0.05),
	}
}

func (fs *FallbackSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// borrowing 1 request/second
	if fs.borrow(fs.clock.Now().Unix()) {
		sd := sdktrace.SamplingResult{
			Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
		}

		sd.Decision = sdktrace.RecordAndSample
		return sd
	}

	// using traceIDRatioBased sampler to sample using 5% fixed rate
	samplingDecision := fs.defaultSampler.ShouldSample(parameters)

	return samplingDecision
}

func (fs *FallbackSampler) Description() string {
	return "fallback sampling with sampling config of 1 req/sec and 5% of additional requests"
}

func (fs *FallbackSampler) borrow(now int64) bool {
	cur := atomic.LoadInt64(&fs.currentEpoch)
	if cur >= now {
		return false
	}
	return atomic.CompareAndSwapInt64(&fs.currentEpoch, cur, now)
}

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
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type FallbackSampler struct {
	currentEpoch   int64
	defaultSampler sdktrace.Sampler
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*FallbackSampler)(nil)

// NewFallbackSampler returns a FallbackSampler which samples 1 req/sec and additional 5% of requests using traceIDRatioBasedSampler.
func NewFallbackSampler() *FallbackSampler {
	return &FallbackSampler{
		defaultSampler: sdktrace.TraceIDRatioBased(0.05),
	}
}

// ShouldSample implements the logic of borrowing 1 req/sec and then use traceIDRatioBasedSampler to sample 5% of additional requests.
func (fs *FallbackSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// borrowing one request every second
	if fs.borrow(time.Now().Unix()) {
		return sdktrace.SamplingResult{
			Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
			Decision:   sdktrace.RecordAndSample,
		}
	}

	// traceIDRatioBasedSampler to sample 5% of additional requests every second
	return fs.defaultSampler.ShouldSample(parameters)
}

// Description returns description of the sampler being used
func (fs *FallbackSampler) Description() string {
	return "FallbackSampler{" + fs.getDescription() + "}"
}

func (fs *FallbackSampler) getDescription() string {
	return "fallback sampling with sampling config of 1 req/sec and 5% of additional requests"
}

func (fs *FallbackSampler) borrow(now int64) bool {
	cur := atomic.LoadInt64(&fs.currentEpoch)
	if cur >= now {
		return false
	}
	return atomic.CompareAndSwapInt64(&fs.currentEpoch, cur, now)
}

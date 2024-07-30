// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray // import "go.opentelemetry.io/contrib/samplers/aws/xray"

import (
	"sync"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// FallbackSampler does the sampling at a rate of 1 req/sec and 5% of additional requests.
type FallbackSampler struct {
	lastTick       time.Time
	quotaBalance   float64
	defaultSampler sdktrace.Sampler
	mu             sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*FallbackSampler)(nil)

// NewFallbackSampler returns a FallbackSampler which samples 1 req/sec and additional 5% of requests using traceIDRatioBasedSampler.
func NewFallbackSampler() *FallbackSampler {
	return &FallbackSampler{
		defaultSampler: sdktrace.TraceIDRatioBased(0.05),
		quotaBalance:   1.0,
	}
}

// ShouldSample implements the logic of borrowing 1 req/sec and then use traceIDRatioBasedSampler to sample 5% of additional requests.
func (fs *FallbackSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// borrowing one request every second
	if fs.take(time.Now(), 1.0) {
		return sdktrace.SamplingResult{
			Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
			Decision:   sdktrace.RecordAndSample,
		}
	}

	// traceIDRatioBasedSampler to sample 5% of additional requests every second
	return fs.defaultSampler.ShouldSample(parameters)
}

// Description returns description of the sampler being used.
func (fs *FallbackSampler) Description() string {
	return "FallbackSampler{fallback sampling with sampling config of 1 req/sec and 5% of additional requests}"
}

// take consumes quota from reservoir, if any remains, then returns true. False otherwise.
func (fs *FallbackSampler) take(now time.Time, itemCost float64) bool { //nolint:unparam
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.lastTick.IsZero() {
		fs.lastTick = now
	}

	if fs.quotaBalance >= itemCost {
		fs.quotaBalance -= itemCost
		return true
	}

	// update quota balance based on elapsed time
	fs.refreshQuotaBalanceLocked(now)

	if fs.quotaBalance >= itemCost {
		fs.quotaBalance -= itemCost
		return true
	}

	return false
}

// refreshQuotaBalanceLocked refreshes the quotaBalance considering elapsedTime.
// It is assumed the lock is held when calling this.
func (fs *FallbackSampler) refreshQuotaBalanceLocked(now time.Time) {
	elapsedTime := now.Sub(fs.lastTick)
	fs.lastTick = now

	// when elapsedTime is higher than 1 even then we need to keep quotaBalance
	// near to 1 so making elapsedTime to 1 for only borrowing 1 per second case
	if elapsedTime.Seconds() > 1.0 {
		fs.quotaBalance += 1.0
	} else {
		// calculate how much credit have we accumulated since the last tick
		fs.quotaBalance += elapsedTime.Seconds()
	}
}

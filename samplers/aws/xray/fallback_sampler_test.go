// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/trace"
)

// assert sampling using fallback sampler.
func TestSampleUsingFallbackSampler(t *testing.T) {
	fs := NewFallbackSampler()
	assert.NotEmpty(t, fs.defaultSampler)
	assert.Equal(t, 1.0, fs.quotaBalance)

	sd := fs.ShouldSample(trace.SamplingParameters{})
	assert.Equal(t, trace.RecordAndSample, sd.Decision)
}

// assert that we only borrow 1 req/sec.
func TestBorrowOnePerSecond(t *testing.T) {
	fs := NewFallbackSampler()
	borrowed := fs.take(time.Unix(1500000000, 0), 1.0)

	// assert that borrowing one per second
	assert.True(t, borrowed)

	borrowed = fs.take(time.Unix(1500000000, 0), 1.0)

	// assert that borrowing again is false during that second
	assert.False(t, borrowed)

	borrowed = fs.take(time.Unix(1500000001, 0), 1.0)

	// assert that borrowing again in next second
	assert.True(t, borrowed)
}

// assert that when elapsedTime is high quotaBalance should still be close to 1.
func TestBorrowWithLargeElapsedTime(t *testing.T) {
	fs := NewFallbackSampler()
	borrowed := fs.take(time.Unix(1500000000, 0), 1.0)

	// assert that borrowing one per second
	assert.True(t, borrowed)

	// Increase the time by 9 seconds
	borrowed = fs.take(time.Unix(1500000009, 0), 1.0)
	assert.True(t, borrowed)
	assert.Equal(t, 0.0, fs.quotaBalance)
}

// assert fallback sampling description.
func TestFallbackSamplerDescription(t *testing.T) {
	fs := NewFallbackSampler()
	s := fs.Description()
	assert.Equal(t, "FallbackSampler{fallback sampling with sampling config of 1 req/sec and 5% of additional requests}", s)
}

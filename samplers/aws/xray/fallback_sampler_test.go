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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/trace"
)

// assert sampling using fallback sampler.
func TestSampleUsingFallbackSampler(t *testing.T) {
	fs := NewFallbackSampler()
	assert.NotEmpty(t, fs.defaultSampler)
	assert.Equal(t, fs.quotaBalance, 1.0)

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
	assert.Equal(t, fs.quotaBalance, 0.0)
}

// assert fallback sampling description.
func TestFallbackSamplerDescription(t *testing.T) {
	fs := NewFallbackSampler()
	s := fs.Description()
	assert.Equal(t, s, "FallbackSampler{fallback sampling with sampling config of 1 req/sec and 5% of additional requests}")
}

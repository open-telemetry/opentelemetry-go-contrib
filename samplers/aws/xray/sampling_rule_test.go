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
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
	"testing"
)

func TestStaleRule(t *testing.T) {
	cr := &centralizedRule{
		matchedRequests: 5,
		reservoir: &centralizedReservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000010)
	assert.True(t, s)
}

func TestFreshRule(t *testing.T) {
	cr := &centralizedRule{
		matchedRequests: 5,
		reservoir: &centralizedReservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000009)
	assert.False(t, s)
}

func TestInactiveRule(t *testing.T) {
	cr := &centralizedRule{
		matchedRequests: 0,
		reservoir: &centralizedReservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000011)
	assert.False(t, s)
}

func TestExpiredReservoirTraceIDRationBasedSample(t *testing.T) {
	// One second past expiration
	clock := &MockClock{
		NowTime: 1500000061,
	}

	// Set random to be within sampling rate
	rand := &MockRand{
		F64: 0.05,
	}

	p := &ruleProperties{
		RuleName: getStringPointer("r1"),
		FixedRate: getFloatPointer(0.06),
	}

	// Expired reservoir
	cr := &centralizedReservoir{
		expiresAt: 1500000060,
		borrowed:  true,
		used:         0,
		capacity:     10,
		currentEpoch: 1500000061,
	}

	csr := &centralizedRule{
		reservoir:  cr,
		ruleProperties: p,
		clock:      clock,
		rand:       rand,
	}

	sd := csr.Sample(trace.SamplingParameters{})

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), csr.sampledRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
}

func TestTakeFromQuotaSample(t *testing.T) {
	clock := &MockClock{
		NowTime: 1500000000,
	}

	cr := &centralizedReservoir{
		quota:     10,
		expiresAt: 1500000060,
		currentEpoch: clock.Now().Unix(),
		used:         0,
	}

	csr := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
		},
		reservoir: cr,
		clock:     clock,
	}

	sd := csr.Sample(trace.SamplingParameters{})

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), csr.sampledRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
	assert.Equal(t, int64(1), csr.reservoir.used)
}

func TestTraceIDRatioBasedSamplerPositive(t *testing.T) {
	clock := &MockClock{
		NowTime: 1500000000,
	}

	// Set random to be within sampling rate
	rand := &MockRand{
		F64: 0.05,
	}

	p := &ruleProperties{
		FixedRate: getFloatPointer(0.06),
		RuleName: getStringPointer("r1"),
	}

	cr := &centralizedReservoir{
		quota:     10,
		expiresAt: 1500000060,
		currentEpoch: clock.Now().Unix(),
		used:         10,
	}

	csr := &centralizedRule{
		reservoir:  cr,
		ruleProperties: p,
		rand:       rand,
		clock:      clock,
	}

	sd := csr.Sample(trace.SamplingParameters{})

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), csr.sampledRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
	assert.Equal(t, int64(10), csr.reservoir.used)
}

func TestSnapshot(t *testing.T) {
	clock := &MockClock{
		NowTime: 1500000000,
	}

	csr := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("rule1"),
		},
		matchedRequests: 100,
		sampledRequests:  12,
		borrowedRequests:  2,
		clock:    clock,
	}

	ss := csr.snapshot()

	// Assert counters were reset
	assert.Equal(t, int64(0), csr.matchedRequests)
	assert.Equal(t, int64(0), csr.sampledRequests)
	assert.Equal(t, int64(0), csr.borrowedRequests)

	// Assert on SamplingStatistics counters
	assert.Equal(t, int64(100), *ss.RequestCount)
	assert.Equal(t, int64(12), *ss.SampledCount)
	assert.Equal(t, int64(2), *ss.BorrowCount)
	assert.Equal(t, "rule1", *ss.RuleName)
	assert.Equal(t, clock.NowTime, *ss.Timestamp)
}
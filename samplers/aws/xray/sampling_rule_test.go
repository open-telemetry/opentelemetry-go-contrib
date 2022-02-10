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

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestStaleRule(t *testing.T) {
	cr := &rule{
		matchedRequests: 5,
		reservoir: &reservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000010)
	assert.True(t, s)
}

func TestFreshRule(t *testing.T) {
	cr := &rule{
		matchedRequests: 5,
		reservoir: &reservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000009)
	assert.False(t, s)
}

func TestInactiveRule(t *testing.T) {
	cr := &rule{
		matchedRequests: 0,
		reservoir: &reservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000011)
	assert.False(t, s)
}

func TestExpiredReservoirTraceIDRationBasedSample(t *testing.T) {
	// One second past expiration
	clock := &mockClock{
		nowTime: 1500000061,
	}

	p := &ruleProperties{
		RuleName:  getStringPointer("r1"),
		FixedRate: getFloatPointer(0.06),
	}

	// Expired reservoir
	cr := &reservoir{
		expiresAt:    1500000060,
		used:         0,
		capacity:     10,
		currentEpoch: 1500000061,
	}

	csr := &rule{
		reservoir:      cr,
		ruleProperties: p,
		clock:          clock,
	}

	csr.Sample(trace.SamplingParameters{})

	assert.Equal(t, int64(0), csr.borrowedRequests)
	assert.Equal(t, int64(1), csr.sampledRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
	assert.Equal(t, int64(0), csr.reservoir.used)
}

func TestExpiredReservoirBorrowSample(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000062,
	}

	p := &ruleProperties{
		RuleName:  getStringPointer("r1"),
		FixedRate: getFloatPointer(0.06),
	}

	// Expired reservoir
	cr := &reservoir{
		expiresAt:    1500000060,
		used:         0,
		capacity:     10,
		currentEpoch: 1500000061,
	}

	csr := &rule{
		reservoir:      cr,
		ruleProperties: p,
		clock:          clock,
	}

	sd := csr.Sample(trace.SamplingParameters{})

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), csr.borrowedRequests)
	assert.Equal(t, int64(0), csr.sampledRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
	assert.Equal(t, int64(0), csr.reservoir.used)
}

func TestTakeFromQuotaSample(t *testing.T) {
	// setting the logger
	newConfig()

	clock := &mockClock{
		nowTime: 1500000000,
	}

	cr := &reservoir{
		quota:        10,
		expiresAt:    1500000060,
		currentEpoch: clock.now().Unix(),
		used:         0,
	}

	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
		},
		reservoir: cr,
		clock:     clock,
	}

	sd := csr.Sample(trace.SamplingParameters{})

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), csr.sampledRequests)
	assert.Equal(t, int64(0), csr.borrowedRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
	assert.Equal(t, int64(1), csr.reservoir.used)
}

func TestTraceIDRatioBasedSampler(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	p := &ruleProperties{
		FixedRate: getFloatPointer(0.06),
		RuleName:  getStringPointer("r1"),
	}

	cr := &reservoir{
		quota:        10,
		expiresAt:    1500000060,
		currentEpoch: clock.now().Unix(),
		used:         10,
	}

	csr := &rule{
		reservoir:      cr,
		ruleProperties: p,
		clock:          clock,
	}

	sd := csr.Sample(trace.SamplingParameters{})

	assert.NotEmpty(t, sd.Decision)
	assert.Equal(t, int64(1), csr.sampledRequests)
	assert.Equal(t, int64(0), csr.borrowedRequests)
	assert.Equal(t, int64(1), csr.matchedRequests)
	assert.Equal(t, int64(10), csr.reservoir.used)
}

func TestSnapshot(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("rule1"),
		},
		matchedRequests:  100,
		sampledRequests:  12,
		borrowedRequests: 2,
		clock:            clock,
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
	assert.Equal(t, clock.nowTime, *ss.Timestamp)
}

func TestAppliesToMatchingWithAllAttrs(t *testing.T) {
	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("rule1"),
			ServiceName: getStringPointer("test-service"),
			ServiceType: getStringPointer("EC2"),
			Host:        getStringPointer("localhost"),
			HTTPMethod:  getStringPointer("GET"),
			URLPath:     getStringPointer("http://127.0.0.1:2000"),
		},
	}

	httpAttrs := []attribute.KeyValue{
		attribute.String("http.host", "localhost"),
		attribute.String("http.method", "GET"),
		attribute.String("http.url", "http://127.0.0.1:2000"),
	}

	params := trace.SamplingParameters{Attributes: httpAttrs}

	assert.True(t, csr.appliesTo(params, "test-service", "EC2"))
}

func TestAppliesToMatchingWithStarHTTPAttrs(t *testing.T) {
	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("rule1"),
			ServiceName: getStringPointer("test-service"),
			ServiceType: getStringPointer("EC2"),
			Host:        getStringPointer("*"),
			HTTPMethod:  getStringPointer("*"),
			URLPath:     getStringPointer("*"),
		},
	}

	assert.True(t, csr.appliesTo(trace.SamplingParameters{}, "test-service", "EC2"))
}

func TestAppliesToMatchingWithHTTPAttrs(t *testing.T) {
	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("rule1"),
			ServiceName: getStringPointer("test-service"),
			ServiceType: getStringPointer("EC2"),
			Host:        getStringPointer("localhost"),
			HTTPMethod:  getStringPointer("GET"),
			URLPath:     getStringPointer("http://127.0.0.1:2000"),
		},
	}

	assert.False(t, csr.appliesTo(trace.SamplingParameters{}, "test-service", "EC2"))
}

func TestAppliesToNoMatching(t *testing.T) {
	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("rule1"),
			ServiceName: getStringPointer("test-service"),
			ServiceType: getStringPointer("EC2"),
			Host:        getStringPointer("*"),
			HTTPMethod:  getStringPointer("*"),
			URLPath:     getStringPointer("*"),
		},
	}

	assert.False(t, csr.appliesTo(trace.SamplingParameters{}, "test-service", "ECS"))
}

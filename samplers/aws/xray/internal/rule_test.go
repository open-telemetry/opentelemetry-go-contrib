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

package internal

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/assert"
)

// assert that rule is active but stale due to quota is expired.
func TestStaleRule(t *testing.T) {
	refreshedAt := time.Unix(1500000000, 0)
	r1 := Rule{
		samplingStatistics: &samplingStatistics{
			matchedRequests: 5,
		},
		reservoir: reservoir{
			refreshedAt: refreshedAt,
			interval:    10,
		},
	}

	now := time.Unix(1500000020, 0)
	s := r1.stale(now)
	assert.True(t, s)
}

// assert that rule is active and not stale.
func TestFreshRule(t *testing.T) {
	refreshedAt := time.Unix(1500000000, 0)
	r1 := Rule{
		samplingStatistics: &samplingStatistics{
			matchedRequests: 5,
		},
		reservoir: reservoir{
			refreshedAt: refreshedAt,
			interval:    10,
		},
	}

	now := time.Unix(1500000009, 0)
	s := r1.stale(now)
	assert.False(t, s)
}

// assert that rule is inactive but not stale.
func TestInactiveRule(t *testing.T) {
	refreshedAt := time.Unix(1500000000, 0)
	r1 := Rule{
		samplingStatistics: &samplingStatistics{
			matchedRequests: 0,
		},
		reservoir: reservoir{
			refreshedAt: refreshedAt,
			interval:    10,
		},
	}

	now := time.Unix(1500000011, 0)
	s := r1.stale(now)
	assert.False(t, s)
}

// assert on snapshot of sampling statistics counters.
func TestSnapshot(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r1",
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  100,
			sampledRequests:  12,
			borrowedRequests: 2,
		},
	}

	now := time.Unix(1500000000, 0)
	ss := r1.snapshot(now)

	// assert counters were reset
	assert.Equal(t, int64(0), r1.samplingStatistics.matchedRequests)
	assert.Equal(t, int64(0), r1.samplingStatistics.sampledRequests)
	assert.Equal(t, int64(0), r1.samplingStatistics.borrowedRequests)

	// assert on SamplingStatistics counters
	assert.Equal(t, int64(100), *ss.RequestCount)
	assert.Equal(t, int64(12), *ss.SampledCount)
	assert.Equal(t, int64(2), *ss.BorrowCount)
	assert.Equal(t, "r1", *ss.RuleName)
}

// assert that reservoir is expired, borrowing 1 req during that second.
func TestExpiredReservoirBorrowSample(t *testing.T) {
	r1 := Rule{
		reservoir: reservoir{
			expiresAt: time.Unix(1500000060, 0),
			capacity:  10,
			mu:        &sync.RWMutex{},
		},
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.06,
		},
		samplingStatistics: &samplingStatistics{},
	}

	now := time.Unix(1500000062, 0)
	sd := r1.Sample(trace.SamplingParameters{}, now)

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), r1.samplingStatistics.borrowedRequests)
	assert.Equal(t, int64(0), r1.samplingStatistics.sampledRequests)
	assert.Equal(t, int64(1), r1.samplingStatistics.matchedRequests)
}

// assert that reservoir is expired, borrowed 1 req during that second so now using traceIDRatioBased sampler.
func TestExpiredReservoirTraceIDRationBasedSample(t *testing.T) {
	r1 := Rule{
		reservoir: reservoir{
			expiresAt: time.Unix(1500000060, 0),
			capacity:  10,
			mu:        &sync.RWMutex{},
			lastTick:  time.Unix(1500000061, 0),
		},
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.06,
		},
		samplingStatistics: &samplingStatistics{},
	}

	now := time.Unix(1500000061, 0)
	sd := r1.Sample(trace.SamplingParameters{}, now)

	assert.NotEmpty(t, sd.Decision)
	assert.Equal(t, int64(0), r1.samplingStatistics.borrowedRequests)
	assert.Equal(t, int64(1), r1.samplingStatistics.sampledRequests)
	assert.Equal(t, int64(1), r1.samplingStatistics.matchedRequests)
}

// assert that reservoir is not expired, quota is available so consuming from quota.
func TestConsumeFromReservoirSample(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r1",
		},
		reservoir: reservoir{
			quota:     10,
			expiresAt: time.Unix(1500000060, 0),
			mu:        &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	now := time.Unix(1500000000, 0)
	sd := r1.Sample(trace.SamplingParameters{}, now)

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
	assert.Equal(t, int64(1), r1.samplingStatistics.sampledRequests)
	assert.Equal(t, int64(0), r1.samplingStatistics.borrowedRequests)
	assert.Equal(t, int64(1), r1.samplingStatistics.matchedRequests)
}

// assert that sampling using traceIDRationBasedSampler when reservoir quota is consumed.
func TestTraceIDRatioBasedSamplerReservoirIsConsumedSample(t *testing.T) {
	r1 := Rule{
		reservoir: reservoir{
			quota:     10,
			expiresAt: time.Unix(1500000060, 0),
			mu:        &sync.RWMutex{},
			lastTick:  time.Unix(1500000000, 0),
		},
		ruleProperties: ruleProperties{
			FixedRate: 0.05,
			RuleName:  "r1",
		},
		samplingStatistics: &samplingStatistics{},
	}

	now := time.Unix(1500000000, 0)
	sd := r1.Sample(trace.SamplingParameters{}, now)

	assert.NotEmpty(t, sd.Decision)
	assert.Equal(t, int64(1), r1.samplingStatistics.sampledRequests)
	assert.Equal(t, int64(0), r1.samplingStatistics.borrowedRequests)
	assert.Equal(t, int64(1), r1.samplingStatistics.matchedRequests)
}

// assert that when fixed rate is 0 traceIDRatioBased sampler will not sample the trace.
func TestTraceIDRatioBasedSamplerFixedRateZero(t *testing.T) {
	r1 := Rule{
		reservoir: reservoir{
			quota:     10,
			expiresAt: time.Unix(1500000060, 0),
			mu:        &sync.RWMutex{},
			lastTick:  time.Unix(1500000000, 0),
		},
		ruleProperties: ruleProperties{
			FixedRate: 0,
			RuleName:  "r1",
		},
		samplingStatistics: &samplingStatistics{},
	}

	now := time.Unix(1500000000, 0)
	sd := r1.Sample(trace.SamplingParameters{}, now)

	assert.Equal(t, sd.Decision, trace.Drop)
}

func TestAppliesToMatchingWithAllAttrs(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "EC2",
			Host:        "localhost",
			HTTPMethod:  "GET",
			URLPath:     "http://127.0.0.1:2000",
		},
	}

	httpAttrs := []attribute.KeyValue{
		attribute.String("http.host", "localhost"),
		attribute.String("http.method", "GET"),
		attribute.String("http.url", "http://127.0.0.1:2000"),
	}

	match, err := r1.appliesTo(trace.SamplingParameters{Attributes: httpAttrs}, "test-service", "EC2")
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that matching will happen when rules has all the HTTP attrs set as '*' and
// span has any attribute values.
func TestAppliesToMatchingWithStarHTTPAttrs(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "EC2",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
		},
	}

	httpAttrs := []attribute.KeyValue{
		attribute.String("http.host", "localhost"),
		attribute.String("http.method", "GET"),
		attribute.String("http.url", "http://127.0.0.1:2000"),
	}

	match, err := r1.appliesTo(trace.SamplingParameters{Attributes: httpAttrs}, "test-service", "EC2")
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that matching will not happen when rules has all the HTTP attrs set as non '*' values and
// span has no HTTP attributes.
func TestAppliesToMatchingWithHTTPAttrsNoSpanAttrs(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "EC2",
			Host:        "localhost",
			HTTPMethod:  "GET",
			URLPath:     "http://127.0.0.1:2000",
		},
	}

	match, err := r1.appliesTo(trace.SamplingParameters{}, "test-service", "EC2")
	require.NoError(t, err)
	assert.False(t, match)
}

// assert that matching will happen when rules has all the HTTP attrs set as '*' values and
// span has no HTTP attributes.
func TestAppliesToMatchingWithStarHTTPAttrsNoSpanAttrs(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "EC2",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
		},
	}

	match, err := r1.appliesTo(trace.SamplingParameters{}, "test-service", "EC2")
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that matching will not happen when rules has some HTTP attrs set as non '*' values and
// span has no HTTP attributes.
func TestAppliesToMatchingWithPartialHTTPAttrsNoSpanAttrs(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "EC2",
			Host:        "*",
			HTTPMethod:  "GET",
			URLPath:     "*",
		},
	}

	match, err := r1.appliesTo(trace.SamplingParameters{}, "test-service", "EC2")
	require.NoError(t, err)
	assert.False(t, match)
}

// assert that matching will not happen when rule and span ServiceType attr value is different.
func TestAppliesToNoMatching(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "EC2",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
		},
	}

	match, err := r1.appliesTo(trace.SamplingParameters{}, "test-service", "ECS")
	require.NoError(t, err)
	assert.False(t, match)
}

// assert that when attribute has http.url is empty, uses http.target wildcard matching.
func TestAppliesToHTTPTargetMatching(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("http.target", "target"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "ECS",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
		},
	}

	match, err := r1.appliesTo(trace.SamplingParameters{Attributes: commonLabels}, "test-service", "ECS")
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that if rules has attribute and span has those attribute with same value then matching will happen.
func TestAttributeMatching(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelB", "raspberry"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelB": "raspberry",
			},
		},
	}

	match, err := r1.attributeMatching(trace.SamplingParameters{Attributes: commonLabels})
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that if rules has no attributes then matching will happen.
func TestAttributeMatchingNoRuleAttrs(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelB", "raspberry"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			Attributes: map[string]string{},
		},
	}

	match, err := r1.attributeMatching(trace.SamplingParameters{Attributes: commonLabels})
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that wildcard attributes will match.
func TestAttributeWildCardMatching(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelB", "raspberry"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			Attributes: map[string]string{
				"labelA": "choco*",
				"labelB": "rasp*",
			},
		},
	}

	match, err := r1.attributeMatching(trace.SamplingParameters{Attributes: commonLabels})
	require.NoError(t, err)
	assert.True(t, match)
}

// assert that if some of the rules attributes are not present in span attributes then matching
// will not happen.
func TestMatchAgainstManifestRulesNoAttributeMatch(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelB", "raspberry"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelC": "fudge",
			},
		},
	}

	match, err := r1.attributeMatching(trace.SamplingParameters{Attributes: commonLabels})
	require.NoError(t, err)
	assert.False(t, match)
}

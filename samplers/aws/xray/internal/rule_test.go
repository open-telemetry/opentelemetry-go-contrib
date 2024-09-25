// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
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
		reservoir: &reservoir{
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
		reservoir: &reservoir{
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
		reservoir: &reservoir{
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
		reservoir: &reservoir{
			expiresAt: time.Unix(1500000060, 0),
			capacity:  10,
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
		reservoir: &reservoir{
			expiresAt: time.Unix(1500000060, 0),
			capacity:  10,
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
		reservoir: &reservoir{
			capacity:  10,
			quota:     10,
			expiresAt: time.Unix(1500000060, 0),
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
		reservoir: &reservoir{
			quota:     10,
			expiresAt: time.Unix(1500000060, 0),
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
		reservoir: &reservoir{
			quota:     10,
			expiresAt: time.Unix(1500000060, 0),
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

	assert.Equal(t, trace.Drop, sd.Decision)
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

// assert early exit when rule properties retrieved from AWS X-Ray console does not match with span attributes.
func TestAppliesToExitEarlyNoMatch(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelC", "fudge"),
	}

	noServiceNameMatch := &Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "local",
			ServiceType: "*",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelC": "fudge",
			},
		},
	}

	noServiceTypeMatch := &Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "*",
			ServiceType: "ECS",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelC": "fudge",
			},
		},
	}

	noHTTPMethodMatcher := &Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "*",
			ServiceType: "*",
			Host:        "*",
			HTTPMethod:  "GET",
			URLPath:     "*",
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelC": "fudge",
			},
		},
	}

	noHTTPHostMatcher := &Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "*",
			ServiceType: "*",
			Host:        "http://localhost:2022",
			HTTPMethod:  "*",
			URLPath:     "*",
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelC": "fudge",
			},
		},
	}

	noHTTPURLPathMatcher := &Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "*",
			ServiceType: "*",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "/test/path",
			Attributes:  map[string]string{},
		},
	}

	noAttributeMatcher := &Rule{
		ruleProperties: ruleProperties{
			RuleName:    "r1",
			ServiceName: "test-service",
			ServiceType: "*",
			Host:        "*",
			HTTPMethod:  "*",
			URLPath:     "*",
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelC": "vanilla",
			},
		},
	}

	tests := []struct {
		rules *Rule
	}{
		{noServiceNameMatch},
		{noServiceTypeMatch},
		{noHTTPMethodMatcher},
		{noHTTPHostMatcher},
		{noHTTPURLPathMatcher},
		{noAttributeMatcher},
	}

	for _, test := range tests {
		match, err := test.rules.appliesTo(trace.SamplingParameters{Attributes: commonLabels}, "test-service", "local")
		require.NoError(t, err)
		require.False(t, match)
	}
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

// validate no data race is happening when updating rule properties and rule targets in manifest while sampling.
func TestUpdatingRulesAndTargetsWhileSamplingConcurrentSafe(t *testing.T) {
	// getSamplingRules response to update existing manifest rule
	ruleRecords := samplingRuleRecords{
		SamplingRule: &ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
	}

	// sampling target document to update existing manifest rule
	rate := 0.05
	quota := float64(10)
	ttl := float64(18000000)
	name := "r1"

	st := samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	// existing rule already present in manifest
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "test",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: &reservoir{
			refreshedAt: time.Unix(18000000, 0),
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  0,
			borrowedRequests: 0,
			sampledRequests:  0,
		},
	}
	clock := &mockClock{
		nowTime: 1500000000,
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	// async rule updates
	go func() {
		for i := 0; i < 100; i++ {
			m.updateRules(&getSamplingRulesOutput{
				SamplingRuleRecords: []*samplingRuleRecords{&ruleRecords},
			})
			time.Sleep(time.Millisecond)
		}
	}()

	// async target updates
	go func() {
		for i := 0; i < 100; i++ {
			manifest := m.deepCopy()

			err := manifest.updateReservoir(&st)
			assert.NoError(t, err)
			time.Sleep(time.Millisecond)

			m.mu.Lock()
			m.Rules = manifest.Rules
			m.mu.Unlock()
		}
	}()

	// sampling logic
	for i := 0; i < 100; i++ {
		_ = r1.Sample(trace.SamplingParameters{}, time.Unix(clock.nowTime+int64(i), 0))
		time.Sleep(time.Millisecond)
	}
}

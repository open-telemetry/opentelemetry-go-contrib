// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"sync/atomic"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Rule represents a sampling rule which contains rule properties and reservoir which keeps tracks of sampling statistics of a rule.
type Rule struct {
	samplingStatistics *samplingStatistics

	// reservoir has equivalent fields to store what we receive from service API getSamplingTargets.
	// https://docs.aws.amazon.com/xray/latest/api/API_GetSamplingTargets.html
	reservoir *reservoir

	// ruleProperty is equivalent to what we receive from service API getSamplingRules.
	// https://docs.aws.amazon.com/cli/latest/reference/xray/get-sampling-rules.html
	ruleProperties ruleProperties
}

type samplingStatistics struct {
	// matchedRequests is the number of requests matched against specific rule.
	matchedRequests int64

	// sampledRequests is the number of requests sampled using specific rule.
	sampledRequests int64

	// borrowedRequests is the number of requests borrowed using specific rule.
	borrowedRequests int64
}

// stale checks if targets (sampling stats) for a given rule is expired or not.
func (r *Rule) stale(now time.Time) bool {
	matchedRequests := atomic.LoadInt64(&r.samplingStatistics.matchedRequests)

	reservoirRefreshTime := r.reservoir.refreshedAt.Add(r.reservoir.interval * time.Second)
	return matchedRequests != 0 && now.After(reservoirRefreshTime)
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *Rule) snapshot(now time.Time) *samplingStatisticsDocument {
	name := r.ruleProperties.RuleName

	matchedRequests := atomic.SwapInt64(&r.samplingStatistics.matchedRequests, int64(0))
	sampledRequests := atomic.SwapInt64(&r.samplingStatistics.sampledRequests, int64(0))
	borrowedRequest := atomic.SwapInt64(&r.samplingStatistics.borrowedRequests, int64(0))

	timeStamp := now.Unix()
	return &samplingStatisticsDocument{
		RequestCount: &matchedRequests,
		SampledCount: &sampledRequests,
		BorrowCount:  &borrowedRequest,
		RuleName:     &name,
		Timestamp:    &timeStamp,
	}
}

// Sample uses sampling targets of a given rule to decide
// which sampling should be done and returns a SamplingResult.
func (r *Rule) Sample(parameters sdktrace.SamplingParameters, now time.Time) sdktrace.SamplingResult {
	sd := sdktrace.SamplingResult{
		Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	atomic.AddInt64(&r.samplingStatistics.matchedRequests, int64(1))

	// Fallback sampling logic if quota for a given rule is expired.
	if r.reservoir.expired(now) {
		// Borrowing one request every second.
		if r.reservoir.take(now, true, 1.0) {
			atomic.AddInt64(&r.samplingStatistics.borrowedRequests, int64(1))

			sd.Decision = sdktrace.RecordAndSample
			return sd
		}

		// Using traceIDRatioBased sampler to sample using fixed rate.
		sd = sdktrace.TraceIDRatioBased(r.ruleProperties.FixedRate).ShouldSample(parameters)

		if sd.Decision == sdktrace.RecordAndSample {
			atomic.AddInt64(&r.samplingStatistics.sampledRequests, int64(1))
		}

		return sd
	}

	// Take from reservoir quota, if quota is available for that second.
	if r.reservoir.take(now, false, 1.0) {
		atomic.AddInt64(&r.samplingStatistics.sampledRequests, int64(1))
		sd.Decision = sdktrace.RecordAndSample

		return sd
	}

	// using traceIDRatioBased sampler to sample using fixed rate
	sd = sdktrace.TraceIDRatioBased(r.ruleProperties.FixedRate).ShouldSample(parameters)

	if sd.Decision == sdktrace.RecordAndSample {
		atomic.AddInt64(&r.samplingStatistics.sampledRequests, int64(1))
	}

	return sd
}

// appliesTo performs a matching against rule properties to see
// if a given rule does match with any of the rule set on AWS X-Ray console.
func (r *Rule) appliesTo(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) (bool, error) {
	var httpTarget string
	var httpURL string
	var httpHost string
	var httpMethod string
	var HTTPURLPathMatcher bool

	if parameters.Attributes != nil {
		for _, attrs := range parameters.Attributes {
			if attrs.Key == "http.target" {
				httpTarget = attrs.Value.AsString()
			}
			if attrs.Key == "http.url" {
				httpURL = attrs.Value.AsString()
			}
			if attrs.Key == "http.host" {
				httpHost = attrs.Value.AsString()
			}
			if attrs.Key == "http.method" {
				httpMethod = attrs.Value.AsString()
			}
		}
	}

	// Attributes and other HTTP span attributes matching.
	attributeMatcher, err := r.attributeMatching(parameters)
	if err != nil {
		return attributeMatcher, err
	}

	if !attributeMatcher {
		return attributeMatcher, nil
	}

	serviceNameMatcher, err := wildcardMatch(r.ruleProperties.ServiceName, serviceName)
	if err != nil {
		return serviceNameMatcher, err
	}

	if !serviceNameMatcher {
		return serviceNameMatcher, nil
	}

	serviceTypeMatcher, err := wildcardMatch(r.ruleProperties.ServiceType, cloudPlatform)
	if err != nil {
		return serviceTypeMatcher, err
	}

	if !serviceTypeMatcher {
		return serviceTypeMatcher, nil
	}

	HTTPMethodMatcher, err := wildcardMatch(r.ruleProperties.HTTPMethod, httpMethod)
	if err != nil {
		return HTTPMethodMatcher, err
	}

	if !HTTPMethodMatcher {
		return HTTPMethodMatcher, nil
	}

	HTTPHostMatcher, err := wildcardMatch(r.ruleProperties.Host, httpHost)
	if err != nil {
		return HTTPHostMatcher, err
	}

	if !HTTPHostMatcher {
		return HTTPHostMatcher, nil
	}

	if httpURL != "" {
		HTTPURLPathMatcher, err = wildcardMatch(r.ruleProperties.URLPath, httpURL)
		if err != nil {
			return HTTPURLPathMatcher, err
		}

		if !HTTPURLPathMatcher {
			return HTTPURLPathMatcher, nil
		}
	} else {
		HTTPURLPathMatcher, err = wildcardMatch(r.ruleProperties.URLPath, httpTarget)
		if err != nil {
			return HTTPURLPathMatcher, err
		}

		if !HTTPURLPathMatcher {
			return HTTPURLPathMatcher, nil
		}
	}

	return true, nil
}

// attributeMatching performs a match on attributes set by users on AWS X-Ray console.
func (r *Rule) attributeMatching(parameters sdktrace.SamplingParameters) (bool, error) {
	match := false
	var err error

	if len(r.ruleProperties.Attributes) == 0 {
		return true, nil
	}

	for key, value := range r.ruleProperties.Attributes {
		unmatchedCounter := 0
		for _, attrs := range parameters.Attributes {
			if key == string(attrs.Key) {
				match, err = wildcardMatch(value, attrs.Value.AsString())
				if err != nil {
					return false, err
				}

				if !match {
					return false, nil
				}
			} else {
				unmatchedCounter++
			}
		}
		if unmatchedCounter == len(parameters.Attributes) {
			return false, nil
		}
	}

	return match, nil
}

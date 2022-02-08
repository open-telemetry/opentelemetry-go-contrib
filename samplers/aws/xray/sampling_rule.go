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
	"sync"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// centralizedRule represents a centralized sampling rule
type rule struct {
	// Centralized reservoir for keeping track of reservoir usage
	reservoir *reservoir

	// sampling rule properties
	ruleProperties *ruleProperties

	// Number of requests matched against this rule
	matchedRequests int64

	// Number of requests sampled using this rule
	sampledRequests int64

	// Number of requests borrowed
	borrowedRequests int64

	// Provides system time
	clock clock

	// Provides random numbers
	rand Rand

	//mu sync.RWMutex
}

// properties is the base set of properties that define a sampling rule.
type ruleProperties struct {
	RuleName      *string            `json:"RuleName"`
	ServiceType   *string            `json:"ServiceType"`
	ResourceARN   *string            `json:"ResourceARN"`
	Attributes    map[string]*string `json:"Attributes"`
	ServiceName   *string            `json:"ServiceName"`
	Host          *string            `json:"Host"`
	HTTPMethod    *string            `json:"HTTPMethod"`
	URLPath       *string            `json:"URLPath"`
	ReservoirSize *int64             `json:"ReservoirSize"`
	FixedRate     *float64           `json:"FixedRate"`
	Priority      *int64             `json:"Priority"`
	Version       *int64             `json:"Version"`
}

// getSamplingRulesInput is used to store
type getSamplingRulesInput struct {
	NextToken *string `json:"NextToken"`
}

type samplingRuleRecords struct {
	SamplingRule *ruleProperties `json:"SamplingRule"`
}

// getSamplingRulesOutput is used to store parsed json sampling rules
type getSamplingRulesOutput struct {
	SamplingRuleRecords []*samplingRuleRecords `json:"SamplingRuleRecords"`
}

// Sample returns SamplingResult with SamplingDecision, TraceState and Attributes
func (r *centralizedRule) Sample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	attributes := []attribute.KeyValue{
		attribute.String("Rule", *r.ruleProperties.RuleName),
	}

	sd := sdktrace.SamplingResult{
		Attributes: attributes,
		Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	now := r.clock.Now().Unix()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.matchedRequests++

	// fallback sampling logic if quota has expired
	if r.reservoir.expired(now) {
		// if reservoir quota is expired then sampling using
		// sampling config: 1 req/sec and then x% of fixedRate

		// Sampling 1 req/sec
		if r.reservoir.borrow(now) {
			globalLogger.Printf(
				"Sampling target has expired for rule %s. Using fallback sampling and borrowing 1 req/sec from reservoir",
				*r.ruleProperties.RuleName,
			)
			r.borrowedRequests++

			sd.Decision = sdktrace.RecordAndSample
			return sd
		}

		globalLogger.Printf(
			"Sampling target has expired for rule %s. Using traceIDRationBased sampler to sample 5 percent of requests during that second",
			*r.ruleProperties.RuleName,
		)

		// using traceIDRatioBased sampler to sample
		samplingDecision := sdktrace.TraceIDRatioBased(*r.ruleProperties.FixedRate).ShouldSample(parameters)

		samplingDecision.Attributes = attributes

		if samplingDecision.Decision == sdktrace.RecordAndSample {
			r.sampledRequests++
		}

		return samplingDecision
	}

	// Take from reservoir quota, if possible
	if r.reservoir.Take(now) {
		r.sampledRequests++
		sd.Decision = sdktrace.RecordAndSample

		return sd
	}

	globalLogger.Printf(
		"Sampling target has been exhausted for rule %s. Using traceIDRatioBased Sampler with fixed rate.",
		*r.ruleProperties.RuleName,
	)

	// using traceIDRatioBased sampler to sample using fixed rate
	samplingDecision := sdktrace.TraceIDRatioBased(*r.ruleProperties.FixedRate).ShouldSample(parameters)

	samplingDecision.Attributes = attributes
	samplingDecision.Tracestate = sd.Tracestate

	if samplingDecision.Decision == sdktrace.RecordAndSample {
		r.sampledRequests++
	}

	return samplingDecision
}

// stale returns true if the quota is due for a refresh. False otherwise.
func (r *centralizedRule) stale(now int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.matchedRequests != 0 && now >= r.reservoir.refreshedAt+r.reservoir.interval
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *centralizedRule) snapshot() *samplingStatisticsDocument {
	r.mu.Lock()

	name := r.ruleProperties.RuleName

	// Copy statistics counters since xraySvc.SamplingStatistics expects
	// pointers to counters, and ours are mutable.
	requests, sampled, borrows := r.matchedRequests, r.sampledRequests, r.borrowedRequests

	// Reset counters
	r.matchedRequests, r.sampledRequests, r.borrowedRequests = 0, 0, 0

	r.mu.Unlock()

	now := r.clock.Now().Unix()
	s := &samplingStatisticsDocument{
		RequestCount: &requests,
		SampledCount: &sampled,
		BorrowCount:  &borrows,
		RuleName:     name,
		Timestamp:    &now,
	}

	return s
}

func (r *centralizedRule) appliesTo(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) bool {
	var httpTarget string
	var httpURL string
	var httpHost string
	var httpMethod string

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

	return (wildcardMatch(*r.ruleProperties.ServiceName, serviceName, true)) &&
		(wildcardMatch(*r.ruleProperties.ServiceType, cloudPlatform, true)) &&
		(wildcardMatch(*r.ruleProperties.Host, httpHost, true)) &&
		(wildcardMatch(*r.ruleProperties.HTTPMethod, httpMethod, true)) &&
		(wildcardMatch(*r.ruleProperties.URLPath, httpURL, true) || wildcardMatch(*r.ruleProperties.URLPath, httpTarget, true))
}

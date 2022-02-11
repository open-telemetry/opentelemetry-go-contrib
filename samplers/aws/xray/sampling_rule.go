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
	"fmt"
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal_xray"
	"sync"
	"sync/atomic"

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
	clock internal_xray.clock

	// Provides random numbers
	rand Rand

	mu sync.RWMutex
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
func (r *rule) Sample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	sd := sdktrace.SamplingResult{
		Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	now := r.clock.now().Unix()
	fmt.Println(r.clock.now())

	atomic.AddInt64(&r.matchedRequests, int64(1))

	// fallback sampling logic if quota has expired
	if r.reservoir.expired(now) {
		// borrowing one request every second
		if r.reservoir.borrow(now) {
			fmt.Println("inside expired reservoir")
			atomic.AddInt64(&r.borrowedRequests, int64(1))

			sd.Decision = sdktrace.RecordAndSample
			return sd
		}

		fmt.Println("inside expired traceIDRatio")
		// using traceIDRatioBased sampler to sample using fixed rate
		sd = sdktrace.TraceIDRatioBased(*r.ruleProperties.FixedRate).ShouldSample(parameters)

		if sd.Decision == sdktrace.RecordAndSample {
			atomic.AddInt64(&r.sampledRequests, int64(1))
		}

		return sd
	}

	// Take from reservoir quota, if possible
	if r.reservoir.Take(now) {
		fmt.Println("inside non expired reservoir")
		atomic.AddInt64(&r.sampledRequests, int64(1))
		sd.Decision = sdktrace.RecordAndSample

		return sd
	}

	fmt.Println("inside non expired traceIDRatio")
	// using traceIDRatioBased sampler to sample using fixed rate
	sd = sdktrace.TraceIDRatioBased(*r.ruleProperties.FixedRate).ShouldSample(parameters)

	if sd.Decision == sdktrace.RecordAndSample {
		r.sampledRequests++
	}

	return sd
}

// stale returns true if the quota is due for a refresh. False otherwise.
func (r *rule) stale(now int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.matchedRequests != 0 && now >= r.reservoir.refreshedAt+r.reservoir.interval
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
//func (r *rule) snapshot() *internal_xray.samplingStatisticsDocument {
//	name := r.ruleProperties.RuleName
//
//	requests, sampled, borrows := r.matchedRequests, r.sampledRequests, r.borrowedRequests
//
//	r.mu.Lock()
//	r.matchedRequests, r.sampledRequests, r.borrowedRequests = 0, 0, 0
//	r.mu.Unlock()
//
//	now := r.clock.now().Unix()
//	return &internal_xray.samplingStatisticsDocument{
//		RequestCount: &requests,
//		SampledCount: &sampled,
//		BorrowCount:  &borrows,
//		RuleName:     name,
//		Timestamp:    &now,
//	}
//}

func (r *rule) appliesTo(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) bool {
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

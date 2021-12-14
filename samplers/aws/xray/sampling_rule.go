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

package main

import (
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"log"
	"sync"
)

// ToDo: other fields will be used in business logic for remote sampling
// centralizedRule represents a centralized sampling rule
type centralizedRule struct {
	// Centralized reservoir for keeping track of reservoir usage
	reservoir *centralizedReservoir

	// sampling rule properties
	ruleProperties *ruleProperties

	// Number of requests matched against this rule
	requests int64
	//
	// Number of requests sampled using this rule
	sampled int64
	//
	// Number of requests borrowed
	borrows int64
	//
	// Timestamp for last match against this rule
	//usedAt int64

	// Provides system time
	clock Clock

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

type samplingRuleRecords struct {
	SamplingRule *ruleProperties `json:"SamplingRule"`
}

// getSamplingRulesOutput is used to store parsed json sampling rules
type getSamplingRulesOutput struct {
	SamplingRuleRecords []*samplingRuleRecords `json:"SamplingRuleRecords"`
}

// Sample returns sdktrace.SamplingResult on whether to sample or not
func (r *centralizedRule) Sample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	attributes := []attribute.KeyValue{
		attribute.String("Rule", *r.ruleProperties.RuleName),
	}

	now := r.clock.Now().Unix()
	sd := sdktrace.SamplingResult{
		Attributes: attributes,
		Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests++

	// Use fallbackSampler if quota has expired
	if r.reservoir.expired(now) {
		//if reservoir quota is expired sampling using fallback sampler
		// sampling config: 1 req/sec and then 5% of additional requests

		// Sampling 1 req/sec
		if r.reservoir.borrow(now) {
			log.Printf(
				"Sampling target has expired for rule %s. Using fallback sampling and borrowing 1 req/sec from reservoir",
				*r.ruleProperties.RuleName,
			)
			r.borrows++

			sd.Decision = sdktrace.RecordAndSample
			return sd
		}

		log.Printf(
			"Sampling target has expired for rule %s. Using fallback sampling and sample 5 percent of requests during that second",
			*r.ruleProperties.RuleName,
		)

		// Sampling using 5% fixed rate sampling if reservoir is consumed for that second
		samplingDecision := bernoulliSample(r.rand, 0.05)
		sd.Decision = samplingDecision

		if sd.Decision == sdktrace.RecordAndSample {
			r.sampled++
		}
		return sd
	}

	// Take from reservoir quota, if possible
	if r.reservoir.Take(now) {
		r.sampled++
		sd.Decision = sdktrace.RecordAndSample

		return sd
	}

	log.Printf(
		"Sampling target has been exhausted for rule %s. Using fixed rate.",
		*r.ruleProperties.RuleName,
	)

	// sampling using bernoulliSample with Fixed Rate given in the rule
	sd.Decision = bernoulliSample(r.rand, *r.ruleProperties.FixedRate)

	if sd.Decision == sdktrace.RecordAndSample {
		r.sampled++
	}

	return sd
}

// bernoulliSample uses bernoulli sampling rate to make a sampling decision
func bernoulliSample(rand Rand, samplingRate float64) sdktrace.SamplingDecision {
	if rand.Float64() < samplingRate {
		return sdktrace.RecordAndSample
	}

	return sdktrace.Drop
}

// stale returns true if the quota is due for a refresh. False otherwise.
func (r *centralizedRule) stale(now int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.requests != 0 && now >= r.reservoir.refreshedAt+r.reservoir.interval
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *centralizedRule) snapshot() *samplingStatisticsDocument {
	r.mu.Lock()

	name := r.ruleProperties.RuleName

	// Copy statistics counters since xraySvc.SamplingStatistics expects
	// pointers to counters, and ours are mutable.
	requests, sampled, borrows := r.requests, r.sampled, r.borrows

	// Reset counters
	r.requests, r.sampled, r.borrows = 0, 0, 0

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

func (r *centralizedRule) AppliesTo() bool {
	// ToDo: Implement matching logic
	return true
}

package internal

import (
	"fmt"
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal/util"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"sync/atomic"
)

// Rule represents a sampling rule which contains rule properties and reservoir which keeps tracks of sampling statistics of a rule
type Rule struct {
	// reservoir has equivalent fields to store what we receive from service API getSamplingTargets
	// https://docs.aws.amazon.com/xray/latest/api/API_GetSamplingTargets.html
	reservoir reservoir

	// equivalent to what we receive from service API getSamplingRules
	// https://docs.aws.amazon.com/cli/latest/reference/xray/get-sampling-rules.html
	ruleProperties ruleProperties

	// number of requests matched against specific rule
	matchedRequests int64

	// number of requests sampled using specific rule
	sampledRequests int64

	// number of requests borrowed using specific rule
	borrowedRequests int64

	clock util.Clock
}

// stale checks if targets (sampling stats) for a given rule is expired or not
func (r *Rule) stale(now int64) bool {
	return r.matchedRequests != 0 && now >= r.reservoir.refreshedAt+r.reservoir.interval
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *Rule) snapshot(now int64) *samplingStatisticsDocument {
	name := r.ruleProperties.RuleName
	requests, sampled, borrowed := r.matchedRequests, r.sampledRequests, r.borrowedRequests

	// reset counters
	r.matchedRequests, r.sampledRequests, r.borrowedRequests = 0, 0, 0

	return &samplingStatisticsDocument{
		RequestCount: &requests,
		SampledCount: &sampled,
		BorrowCount:  &borrowed,
		RuleName:     &name,
		Timestamp:    &now,
	}
}

// Sample uses sampling targets of a given rule to decide which sampling should be done and returns a SamplingResult.
func (r *Rule) Sample(parameters sdktrace.SamplingParameters, now int64) sdktrace.SamplingResult {
	sd := sdktrace.SamplingResult{
		Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	atomic.AddInt64(&r.matchedRequests, int64(1))

	// fallback sampling logic if quota for a given rule is expired
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
		sd = sdktrace.TraceIDRatioBased(r.ruleProperties.FixedRate).ShouldSample(parameters)

		if sd.Decision == sdktrace.RecordAndSample {
			atomic.AddInt64(&r.sampledRequests, int64(1))
		}

		return sd
	}

	// Take from reservoir quota, if quota is available for that second
	if r.reservoir.take(now) {
		fmt.Println("inside non expired reservoir")
		atomic.AddInt64(&r.sampledRequests, int64(1))
		sd.Decision = sdktrace.RecordAndSample

		return sd
	}

	fmt.Println("inside non expired traceIDRatio")
	// using traceIDRatioBased sampler to sample using fixed rate
	sd = sdktrace.TraceIDRatioBased(r.ruleProperties.FixedRate).ShouldSample(parameters)

	if sd.Decision == sdktrace.RecordAndSample {
		atomic.AddInt64(&r.sampledRequests, int64(1))
	}

	return sd
}

// appliesTo performs a matching against rule properties to see if a given rule does match with any of the rule set on AWS X-Ray console
func (r *Rule) appliesTo(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) bool {
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

	return (r.attributeMatching(parameters)) && (wildcardMatch(r.ruleProperties.ServiceName, serviceName)) &&
		(wildcardMatch(r.ruleProperties.ServiceType, cloudPlatform)) &&
		(wildcardMatch(r.ruleProperties.Host, httpHost)) &&
		(wildcardMatch(r.ruleProperties.HTTPMethod, httpMethod)) &&
		(wildcardMatch(r.ruleProperties.URLPath, httpURL) || wildcardMatch(r.ruleProperties.URLPath, httpTarget))
}

// attributeMatching performs a match on attributes set by users on AWS X-Ray console
func (r *Rule) attributeMatching(parameters sdktrace.SamplingParameters) bool {
	match := false
	if len(r.ruleProperties.Attributes) > 0 {
		for key, value := range r.ruleProperties.Attributes {
			for _, attrs := range parameters.Attributes {
				if key == string(attrs.Key) {
					match = wildcardMatch(value, attrs.Value.AsString())
				} else {
					match = false
				}
			}
		}
		return match
	}

	return true
}
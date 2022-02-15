package internal_xray

import (
	"fmt"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"sync"
	"sync/atomic"
)

// centralizedRule represents a centralized sampling rule
type rule struct {
	// Centralized reservoir for keeping track of reservoir usage
	reservoir reservoir

	// sampling rule properties
	ruleProperties ruleProperties

	// Number of requests matched against this rule
	matchedRequests int64

	// Number of requests sampled using this rule
	sampledRequests int64

	// Number of requests borrowed
	borrowedRequests int64

	mu sync.RWMutex
}

// properties is the base set of properties that define a sampling rule.
type ruleProperties struct {
	RuleName      string            `json:"RuleName"`
	ServiceType   string            `json:"ServiceType"`
	ResourceARN   string            `json:"ResourceARN"`
	Attributes    map[string]string `json:"Attributes"`
	ServiceName   string            `json:"ServiceName"`
	Host          string            `json:"Host"`
	HTTPMethod    string            `json:"HTTPMethod"`
	URLPath       string            `json:"URLPath"`
	ReservoirSize int64             `json:"ReservoirSize"`
	FixedRate     float64           `json:"FixedRate"`
	Priority      int64             `json:"Priority"`
	Version       int64             `json:"Version"`
}

func (r *rule) stale(now int64) bool {
	r.mu.RLock()
	r.mu.RUnlock()

	return r.matchedRequests != 0 && now >= r.reservoir.refreshedAt+r.reservoir.interval
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *rule) snapshot() *samplingStatisticsDocument {
	clock := &defaultClock{}
	now := clock.now().Unix()

	r.mu.RLock()
	name := r.ruleProperties.RuleName
	requests, sampled, borrowed := r.matchedRequests, r.sampledRequests, r.borrowedRequests
	r.mu.RUnlock()

	// reset counters
	atomic.CompareAndSwapInt64(&r.matchedRequests, requests, int64(0))
	atomic.CompareAndSwapInt64(&r.sampledRequests, sampled, int64(0))
	atomic.CompareAndSwapInt64(&r.borrowedRequests, borrowed, int64(0))

	requests, sampled, borrowed = 4, 4, 0

	return &samplingStatisticsDocument{
		RequestCount: &requests,
		SampledCount: &sampled,
		BorrowCount:  &borrowed,
		RuleName:     &name,
		Timestamp:    &now,
	}
}

func (r *rule) Sample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	sd := sdktrace.SamplingResult{
		Tracestate: trace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	clock := &defaultClock{}
	now := clock.now().Unix()

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
		sd = sdktrace.TraceIDRatioBased(r.ruleProperties.FixedRate).ShouldSample(parameters)

		if sd.Decision == sdktrace.RecordAndSample {
			atomic.AddInt64(&r.sampledRequests, int64(1))
		}

		return sd
	}

	// Take from reservoir quota, if possible
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

func (r *rule) AppliesTo(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) bool {
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

	return (wildcardMatch(r.ruleProperties.ServiceName, serviceName, true)) &&
		(wildcardMatch(r.ruleProperties.ServiceType, cloudPlatform, true)) &&
		(wildcardMatch(r.ruleProperties.Host, httpHost, true)) &&
		(wildcardMatch(r.ruleProperties.HTTPMethod, httpMethod, true)) &&
		(wildcardMatch(r.ruleProperties.URLPath, httpURL, true) || wildcardMatch(r.ruleProperties.URLPath, httpTarget, true))
}

// wildcardMatch returns true if text matches pattern at the given case-sensitivity; returns false otherwise.
func wildcardMatch(pattern, text string, caseInsensitive bool) bool {
	patternLen := len(pattern)
	textLen := len(text)
	if patternLen == 0 {
		return textLen == 0
	}

	if pattern == "*" {
		return true
	}

	if caseInsensitive {
		pattern = strings.ToLower(pattern)
		text = strings.ToLower(text)
	}

	i := 0
	p := 0
	iStar := textLen
	pStar := 0

	for i < textLen {
		if p < patternLen {
			switch pattern[p] {
			case text[i]:
				i++
				p++
				continue
			case '?':
				i++
				p++
				continue
			case '*':
				iStar = i
				pStar = p
				p++
				continue
			}
		}
		if iStar == textLen {
			return false
		}
		iStar++
		i = iStar
		p = pStar + 1
	}

	for p < patternLen && pattern[p] == '*' {
		p++
	}

	return p == patternLen && i == textLen
}
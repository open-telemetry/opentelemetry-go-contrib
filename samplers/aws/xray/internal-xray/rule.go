package internal_xray

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"sync"
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
	return false
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *rule) snapshot() *samplingStatisticsDocument {
	return nil
}

func (r *rule) Sample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	return sdktrace.SamplingResult{}
}

func (r *rule) AppliesTo(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) bool {
	return false
}
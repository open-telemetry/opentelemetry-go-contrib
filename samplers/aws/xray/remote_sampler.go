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
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

// Sampler decides whether a trace should be sampled and exported.
type Sampler interface {
	// ToDo: add SamplingResult and SamplingParameters when adding business logic for centralized sampling
	// ShouldSample returns a SamplingResult based on a decision made from the
	// passed parameters.
	ShouldSample()
	// Description returns information describing the Sampler.
	Description() string
}

// ToDo: take input from customer as an option when calling RemoteSampler API
//type RemoteSamplerConfig struct {
//	// collector ProxyEndpoint to call GetSamplingRules and GetSamplingTargets APIs
//	ProxyEndpoint string
//	// PollingInterval (seconds) to retrieve sampling rules from AWS X-Ray Console
//	PollingInterval int64
//}

// RemoteSampler is an implementation of SamplingStrategy.
type RemoteSampler struct {
	// List of known centralized sampling rules
	manifest *centralizedManifest

	// proxy is used for getting quotas and sampling rules
	proxy *proxy

	// pollerStart, if true represents rule and target pollers are started
	pollerStart bool

	// Provides system time
	clock Clock

	mu sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ Sampler = (*RemoteSampler)(nil)

// NewRemoteSampler returns a centralizedSampler which decides to sample a given request or not.
func NewRemoteSampler() *RemoteSampler {
	return newRemoteSampler()
}

func newRemoteSampler() *RemoteSampler {
	clock := &DefaultClock{}

	m := &centralizedManifest{
		Rules: []*centralizedRule{},
		Index: map[string]*centralizedRule{},
		clock: clock,
	}

	return &RemoteSampler{
		pollerStart: false,
		clock:       clock,
		manifest:    m,
	}
}

func (rs *RemoteSampler) ShouldSample() {
	// ToDo: add business logic for remote sampling
	rs.mu.Lock()
	if !rs.pollerStart {
		rs.start()
	}
	rs.mu.Unlock()
}

func (rs *RemoteSampler) Description() string {
	return "remote sampling with AWS X-Ray"
}

func (rs *RemoteSampler) start() {
	if !rs.pollerStart {
		var er error
		// ToDo: add config to set proxy value
		rs.proxy, er = newProxy("127.0.0.1:2000")
		if er != nil {
			panic(er)
		}
		rs.startRulePoller()
	}

	rs.pollerStart = true
}

func (rs *RemoteSampler) startRulePoller() {
	// ToDo: Add logic to do periodic sampling rules call via background goroutines.
	// Period = 300s, Jitter = 5s
	t := NewTimer(10*time.Second, 5*time.Second)

	for range t.C() {
		t.Reset()
		if err := rs.refreshManifest(); err != nil {
			log.Printf("Error occurred while refreshing sampling rules. %v\n", err)
		} else {
			log.Println("Successfully fetched sampling rules")
		}
	}
}

func (rs *RemoteSampler) refreshManifest() (err error) {
	// Explicitly recover from panics since this is the entry point for a long-running goroutine
	// and we can not allow a panic to propagate to the application code.
	defer func() {
		if r := recover(); r != nil {
			// Resort to bring rules array into consistent state.
			//cs.manifest.sort()

			err = fmt.Errorf("%v", r)
		}
	}()

	// Compute 'now' before calling GetSamplingRules to avoid marking manifest as
	// fresher than it actually is.
	now := rs.clock.Now().Unix()

	// Get sampling rules from proxy
	records, err := rs.proxy.getSamplingRules()
	if err != nil {
		return
	}

	// Set of rules to exclude from pruning
	actives := map[*centralizedRule]bool{}

	// Create missing rules. Update existing ones.
	failed := false

	switch x := records.(type) {
	case []interface{}:
		for _, e := range x {
			svcRule := e.(map[string]interface{})["SamplingRule"].(map[string]interface{})

			ruleProperties := &properties{
				ruleName:      svcRule["RuleName"].(string),
				serviceType:   svcRule["ServiceType"].(string),
				resourceARN:   svcRule["ResourceARN"].(string),
				attributes:    svcRule["Attributes"],
				serviceName:   svcRule["ServiceName"].(string),
				host:          svcRule["Host"].(string),
				httpMethod:    svcRule["HTTPMethod"].(string),
				urlPath:       svcRule["URLPath"].(string),
				reservoirSize: int64(svcRule["ReservoirSize"].(float64)),
				fixedRate:     svcRule["FixedRate"].(float64),
				priority:      int64(svcRule["Priority"].(float64)),
				version:       int64(svcRule["Version"].(float64)),
			}

			if svcRule == nil {
				log.Println("Sampling rule missing from sampling rule record.")
				failed = true
				continue
			}

			if ruleProperties.ruleName == "" {
				log.Println("Sampling rule without rule name is not supported")
				failed = true
				continue
			}

			// Only sampling rule with version 1 is valid
			if ruleProperties.version == 0 {
				log.Println("Sampling rule without version number is not supported: ", ruleProperties.ruleName)
				failed = true
				continue
			}

			if ruleProperties.version != int64(1) {
				log.Println("Sampling rule without version 1 is not supported: ", ruleProperties.ruleName)
				failed = true
				continue
			}

			if reflect.ValueOf(ruleProperties.attributes).Len() != 0 {
				log.Println("Sampling rule with non nil Attributes is not applicable: ", ruleProperties.ruleName)
				continue
			}

			if ruleProperties.resourceARN == "" {
				log.Println("Sampling rule without ResourceARN is not applicable: ", ruleProperties.ruleName)
				continue
			}

			if ruleProperties.resourceARN != "*" {
				log.Println("Sampling rule with ResourceARN not equal to * is not applicable: ", ruleProperties.ruleName)
				continue
			}

			// Create/update rule
			r, putErr := rs.manifest.putRule(ruleProperties)
			if putErr != nil {
				failed = true
				log.Printf("Error occurred creating/updating rule. %v\n", putErr)
			} else if r != nil {
				actives[r] = true
			}
		}
	default:
		log.Printf("unhandled type: %T\n", records)
	}

	// Set err if updates failed
	if failed {
		err = errors.New("error occurred creating/updating rules")
	}

	// Prune inactive rules
	rs.manifest.prune(actives)

	// Re-sort to fix matching priorities
	rs.manifest.sort()

	// Update refreshedAt timestamp
	rs.manifest.mu.Lock()
	rs.manifest.refreshedAt = now
	rs.manifest.mu.Unlock()

	return
}

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
	"context"
	crypto "crypto/rand"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

// RemoteSampler is an implementation of SamplingStrategy.
type RemoteSampler struct {
	// List of known centralized sampling rules
	manifest *centralizedManifest

	// proxy is used for getting quotas and sampling rules
	xrayClient *xrayClient

	// pollerStart, if true represents rule and target pollers are started
	pollerStart bool

	// Unique ID used by XRay service to identify this client
	clientID string

	// Provides system time
	clock Clock

	mu sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ trace.Sampler = (*RemoteSampler)(nil)

// NewRemoteSampler returns a centralizedSampler which decides to sample a given request or not.
func NewRemoteSampler() *RemoteSampler {
	// Generate clientID
	var r [12]byte

	_, err := crypto.Read(r[:])
	if err != nil {
		log.Println("error reading cryptographically secure random number generator")
		return nil
	}

	id := fmt.Sprintf("%02x", r)

	clock := &DefaultClock{}

	m := &centralizedManifest{
		rules: []*centralizedRule{},
		index: map[string]*centralizedRule{},
		clock: clock,
	}

	return &RemoteSampler{
		pollerStart: false,
		clock:       clock,
		manifest:    m,
		clientID:    id,
		// ToDo: take proxyEndpoint and pollingInterval from users
		xrayClient: newClient("127.0.0.1:2000"),
	}
}

func (rs *RemoteSampler) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	// ToDo: add business logic for remote sampling
	rs.mu.Lock()
	if !rs.pollerStart {
		rs.start()
	}
	rs.mu.Unlock()

	return trace.SamplingResult{}
}

func (rs *RemoteSampler) Description() string {
	return "remote sampling with AWS X-Ray"
}

func (rs *RemoteSampler) start() {
	if !rs.pollerStart {
		rs.startRulePoller()
	}

	rs.pollerStart = true
}

func (rs *RemoteSampler) startRulePoller() {
	// Initial refresh
	go func() {
		if err := rs.refreshManifest(); err != nil {
			log.Printf("Error occurred while refreshing sampling rules. %v\n", err)
		} else {
			log.Println("Successfully fetched sampling rules")
		}
	}()

	// Periodic manifest refresh
	go func() {
		// Period = 300s, Jitter = 5s
		t := newTicker(300*time.Second, 5*time.Second)

		for range t.C() {
			if err := rs.refreshManifest(); err != nil {
				log.Printf("Error occurred while refreshing sampling rules. %v\n", err)
			} else {
				log.Println("Successfully fetched sampling rules")
			}
		}
	}()
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
	rules, err := rs.xrayClient.getSamplingRules(context.Background())
	if err != nil {
		return
	}

	// Set of rules to exclude from pruning
	actives := map[centralizedRule]bool{}

	// Create missing rules. Update existing ones.
	failed := false

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == nil {
			log.Println("Sampling rule without rule name is not supported")
			failed = true
			continue
		}

		// Only sampling rule with version 1 is valid
		if records.SamplingRule.Version == nil {
			log.Println("Sampling rule without version number is not supported: ", *records.SamplingRule.RuleName)
			failed = true
			continue
		}

		if *records.SamplingRule.Version != int64(1) {
			log.Println("Sampling rule without version 1 is not supported: ", *records.SamplingRule.RuleName)
			failed = true
			continue
		}

		if len(records.SamplingRule.Attributes) != 0 {
			log.Println("Sampling rule with non nil Attributes is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ResourceARN == nil {
			log.Println("Sampling rule without ResourceARN is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		if *records.SamplingRule.ResourceARN != "*" {
			log.Println("Sampling rule with ResourceARN not equal to * is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		// Create/update rule
		r, putErr := rs.manifest.putRule(records.SamplingRule)
		if putErr != nil {
			failed = true
			log.Printf("Error occurred creating/updating rule. %v\n", putErr)
		} else if r != nil {
			actives[*r] = true
		}
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

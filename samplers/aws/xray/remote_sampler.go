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
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal_xray"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// remoteSampler is a sampler for AWS X-Ray which polls sampling rules and sampling targets
// to make a sampling decision based on rules set by users on AWS X-Ray console
type remoteSampler struct {
	// manifest is the list of known centralized sampling rules.
	manifest internal_xray.Manifest

	// xrayClient is used for getting quotas and sampling rules.
	xrayClient *internal_xray.XrayClient

	// pollerStarted, if true represents rule and target pollers are started.
	pollerStarted bool

	// samplingRulesPollingInterval, default is 300 seconds.
	samplingRulesPollingInterval time.Duration

	// matching attribute
	serviceName string

	// matching attribute
	cloudPlatform string

	// Unique ID used by XRay service to identify this client
	clientID string

	// fallback sampler
	fallbackSampler *FallbackSampler

	// logger for logging
	logger logr.Logger

	mu sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*remoteSampler)(nil)

// NewRemoteSampler returns a sampler which decides to sample a given request or not
// based on the sampling rules set by users on AWS X-Ray console. Sampler also periodically polls
// sampling rules and sampling targets.
func NewRemoteSampler(ctx context.Context, serviceName string, cloudPlatform string, opts ...Option) (sdktrace.Sampler, error) {
	cfg := newConfig(opts...)

	// validate config
	err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Generate clientID
	var r [12]byte

	_, err = crypto.Read(r[:])
	if err != nil {
		return nil, fmt.Errorf("unable to generate client ID: %w", err)
	}

	id := fmt.Sprintf("%02x", r)

	client, err := internal_xray.NewClient(cfg.endpoint)
	if err != nil {
		return nil, err
	}

	remoteSampler := &remoteSampler{
		manifest:                     internal_xray.Manifest{},
		xrayClient:                   client,
		clientID:                     id,
		samplingRulesPollingInterval: cfg.samplingRulesPollingInterval,
		fallbackSampler:              NewFallbackSampler(),
		serviceName:                  serviceName,
		cloudPlatform:                cloudPlatform,
		logger:                       cfg.logger,
	}

	// starts the rule and target poller
	remoteSampler.start(ctx)

	return remoteSampler, nil
}

func (rs *remoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	rs.mu.RLock()
	m := rs.manifest
	rs.mu.RUnlock()

	// Use fallback sampler when manifest is expired
	if m.expired() {
		rs.logger.V(5).Info("Centralized manifest expired. Using fallback sampling strategy")
		return rs.fallbackSampler.ShouldSample(parameters)
	}

	// Match against known rules
	for _, r := range m.rules {
		applicable := r.appliesTo(parameters, rs.serviceName, rs.cloudPlatform)

		if applicable {
			rs.logger.V(5).Info("Applicable rule", "RuleName", *r.ruleProperties.RuleName)

			return r.Sample(parameters)
		}
	}

	// Use fallback sampler when request does not match against known rules
	rs.logger.V(5).Info("No match against centralized rules using fallback sampling strategy")
	return rs.fallbackSampler.ShouldSample(parameters)
}

func (rs *remoteSampler) Description() string {
	return "AwsXrayRemoteSampler{" + rs.getDescription() + "}"
}

func (rs *remoteSampler) getDescription() string {
	return "remote sampling with AWS X-Ray"
}

func (rs *remoteSampler) start(ctx context.Context) {
	if !rs.pollerStarted {
		rs.pollerStarted = true
		rs.startPoller(ctx)
	}
}

// startPoller starts the rule and target poller in a separate go routine which runs periodically to refresh manifest and
// targets
func (rs *remoteSampler) startPoller(ctx context.Context) {
	go func() {
		// Period = 300s, Jitter = 5s
		rulesTicker := newTicker(rs.samplingRulesPollingInterval, 5*time.Second)

		// Period = 10.1s, Jitter = 100ms
		targetTicker := newTicker(5*time.Second+100*time.Millisecond, 100*time.Millisecond)

		for {
			select {
			case _, more := <-rulesTicker.C():
				if !more {
					return
				}

				// fetch sampling rules
				if err := rs.manifest.RefreshManifest(ctx, rs.xrayClient, rs.logger); err != nil {
					rs.logger.Error(err, "Error occurred while refreshing sampling rules")
				} else {
					rs.logger.V(5).Info("Successfully fetched sampling rules")
				}
				continue
			case _, more := <-targetTicker.C():
				if !more {
					return
				}

				// fetch sampling targets
				if err := rs.manifest.RefreshTargets(ctx, rs.xrayClient, rs.logger); err != nil {
					rs.logger.Error(err, "Error occurred while refreshing targets for sampling rules")
				}
				continue
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (rs *remoteSampler) refreshManifest(ctx context.Context) (err error) {

}

// refreshTargets refreshes targets for sampling rules. It calls the XRay service proxy with sampling
// statistics for the previous interval and receives targets for the next interval.
func (rs *remoteSampler) refreshTargets(ctx context.Context) (err error) {
	// Flag indicating batch failure
	failed := false

	// Flag indicating whether or not manifest should be refreshed
	refresh := false

	// Generate sampling statistics
	statistics := rs.snapshots()

	// Do not refresh targets if no statistics to report
	if len(statistics) == 0 {
		rs.logger.V(5).Info("No statistics to report and not refreshing sampling targets")
		return nil
	}

	// Get sampling targets
	output, err := rs.xrayClient.getSamplingTargets(ctx, statistics)
	if err != nil {
		return fmt.Errorf("refreshTargets: Error occurred while getting sampling targets: %w", err)
	}

	// Update sampling targets
	for _, t := range output.SamplingTargetDocuments {
		if err = rs.updateTarget(t); err != nil {
			failed = true
			rs.logger.Error(err, "Error occurred updating target for rule")
		}
	}

	// Consume unprocessed statistics messages
	for _, s := range output.UnprocessedStatistics {
		rs.logger.V(5).Info(
			"Error occurred updating sampling target for rule, code and message", "RuleName", *s.RuleName, "ErrorCode",
			*s.ErrorCode,
			"Message", *s.Message,
		)

		// Do not set any flags if error is unknown
		if s.ErrorCode == nil || s.RuleName == nil {
			continue
		}

		// Set batch failure if any sampling statistics return 5xx
		if strings.HasPrefix(*s.ErrorCode, "5") {
			failed = true
		}

		// Set refresh flag if any sampling statistics return 4xx
		if strings.HasPrefix(*s.ErrorCode, "4") {
			refresh = true
		}
	}

	// Set err if updates failed
	if failed {
		err = errors.New("error occurred updating sampling targets")
	} else {
		rs.logger.V(5).Info("Successfully refreshed sampling targets")
	}

	// Set refresh flag if modifiedAt timestamp from remote is greater than ours.
	if remote := output.LastRuleModification; remote != nil {
		rs.mu.RLock()
		local := rs.manifest.refreshedAt
		rs.mu.RUnlock()

		if int64(*remote) >= local {
			refresh = true
		}
	}

	// Perform out-of-band async manifest refresh if flag is set
	if refresh {
		rs.logger.V(5).Info("Refreshing sampling rules out-of-band")

		go func() {
			if err := rs.refreshManifest(ctx); err != nil {
				rs.logger.Error(err, "Error occurred refreshing sampling rules out-of-band")
			}
		}()
	}

	return
}

// samplingStatistics takes a snapshot of sampling statistics from all rules, resetting
// statistics counters in the process.
//func (rs *remoteSampler) snapshots() []*internal_xray.samplingStatisticsDocument {
//	rs.mu.RLock()
//	m := rs.manifest
//	rs.mu.RUnlock()
//
//	now := rs.clock.now().Unix()
//
//	statistics := make([]*internal_xray.samplingStatisticsDocument, 0, len(rs.manifest.rules)+1)
//
//	// Generate sampling statistics for user-defined rules
//	for _, r := range m.rules {
//		if r.stale(now) {
//			s := r.snapshot()
//			s.ClientID = &rs.clientID
//
//			statistics = append(statistics, s)
//		}
//	}
//
//	return statistics
//}

// updateTarget updates sampling targets for the rule specified in the target struct.
//func (rs *remoteSampler) updateTarget(t *internal_xray.samplingTargetDocument) (err error) {
//	// Pre-emptively dereference xraySvc.SamplingTarget fields and return early on nil values
//	// A panic in the middle of an update may leave the rule in an inconsistent state.
//	if t.RuleName == nil {
//		return errors.New("invalid sampling target. Missing rule name")
//	}
//
//	if t.FixedRate == nil {
//		return fmt.Errorf("invalid sampling target for rule %s. Missing fixed rate", *t.RuleName)
//	}
//
//	// Rule for given target
//	rs.mu.RLock()
//	r, ok := rs.manifest.index[*t.RuleName]
//	rs.mu.RUnlock()
//
//	if !ok {
//		return fmt.Errorf("rule %s not found", *t.RuleName)
//	}
//
//	r.mu.Lock()
//	defer r.mu.Unlock()
//
//	r.reservoir.refreshedAt = rs.clock.now().Unix()
//
//	// Update non-optional attributes from response
//	*r.ruleProperties.FixedRate = *t.FixedRate
//
//	// Update optional attributes from response
//	if t.ReservoirQuota != nil {
//		r.reservoir.quota = *t.ReservoirQuota
//	}
//	if t.ReservoirQuotaTTL != nil {
//		r.reservoir.expiresAt = int64(*t.ReservoirQuotaTTL)
//	}
//	if t.Interval != nil {
//		r.reservoir.interval = *t.Interval
//	}
//
//	return nil
//}

func main() {
	ctx := context.Background()
	rs, _ := NewRemoteSampler(ctx, "test", "test-platform")

	for i := 0; i < 1000; i++ {
		rs.ShouldSample(sdktrace.SamplingParameters{})
		time.Sleep(250 * time.Millisecond)
	}
}

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

package push

import (
	"sync"
	"time"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric"
)

const tolerance float64 = 0.1

// PeriodMatcher pairs metric names to their associated periods and collection
// information. Its purpose is to build a Rule, used by the Accumulator when
// determining which metrics to collect.
type PeriodMatcher struct {
	metrics   map[string]*collectData
	startTime time.Time

	m     sync.Mutex
	sched []*pb.MetricConfigResponse_Schedule
}

type collectData struct {
	lastCollected time.Time
	period        time.Duration
}

// MarkStart records the starting time for the PeriodMatcher. Its purpose is
// to align the metric collection schedule to a particular starting point. If
// unset, then all metrics will be collected on the first collection sweep.
func (matcher *PeriodMatcher) MarkStart(startTime time.Time) {
	matcher.startTime = startTime
}

// ApplySchedules sets the schedules that a PeriodMatcher consults when
// constructing a Rule. After processing the schedules, ApplySchedules returns
// the optimal period with which a controller should run a collection sweep.
//
// This function may be called concurrently.
func (matcher *PeriodMatcher) ApplySchedules(sched []*pb.MetricConfigResponse_Schedule) time.Duration {
	matcher.m.Lock()
	matcher.sched = sched
	matcher.metrics = make(map[string]*collectData)
	matcher.m.Unlock()

	return getExportPeriod(matcher.sched)
}

// TODO: handle explicit zeros
func getExportPeriod(sched []*pb.MetricConfigResponse_Schedule) time.Duration {
	if len(sched) == 0 {
		panic("matcher has not applied any schedules")
	}

	checkPeriod := sched[0].PeriodSec
	for _, schedule := range sched[1:] {
		checkPeriod = gcd(checkPeriod, schedule.PeriodSec)
	}

	return time.Duration(checkPeriod) * time.Second
}

// Euclid's algorithm
func gcd(a, b int32) int32 {
	if a < b {
		return gcd(b, a)
	}

	if a == 0 {
		panic("cannot find GCD of zero values")
	}

	if b == 0 {
		return a
	}

	return gcd(b, a%b)
}

// BuildRule constructs a Rule function. This function can then be passed to
// the Accumulator to decide which metrics should be collected in the
// current collection sweep, based on the time passed to this function.
func (matcher *PeriodMatcher) BuildRule(now time.Time) metric.Rule {
	return func(name string) bool {
		matcher.m.Lock()
		defer matcher.m.Unlock()

		data, ok := matcher.metrics[name]
		if !ok {
			matcher.metrics[name] = &collectData{
				lastCollected: matcher.startTime,
				period:        matcher.matchPeriod(name),
			}

			data = matcher.metrics[name]
		}

		if data.period == 0 {
			return false
		}

		var doCollect bool
		boundary := (1 - tolerance) * float64(data.period)
		nextCollection := data.lastCollected.Add(time.Duration(boundary))
		if now.After(nextCollection) {
			data.lastCollected = now
			doCollect = true
		}

		return doCollect
	}
}

func (matcher *PeriodMatcher) matchPeriod(name string) time.Duration {
	var minPeriod int32
	for _, schedule := range matcher.sched {
		if metricpattern.Matches(name, schedule.InclusionPatterns) &&
			!metricpattern.Matches(name, schedule.ExclusionPatterns) &&
			(minPeriod == 0 || minPeriod > schedule.PeriodSec) {

			minPeriod = schedule.PeriodSec
		}
	}

	return time.Duration(minPeriod) * time.Second
}

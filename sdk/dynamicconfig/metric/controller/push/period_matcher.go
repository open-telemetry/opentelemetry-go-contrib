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

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric"
)

const tolerance float64 = 0.1

// TODO: add PeriodMatcher to pusher and accumulator, remove dynamicextension
type PeriodMatcher struct {
	metrics map[string]*CollectData

	m     sync.Mutex
	sched []*pb.MetricConfigResponse_Schedule
}

type CollectData struct {
	lastCollected time.Time
	period        time.Duration
}

func (matcher *PeriodMatcher) BuildRule(now time.Time) metric.Rule {
	return func(name string) bool {
		matcher.m.Lock()
		defer matcher.m.Unlock()

		var doCollect bool
		if data, ok := matcher.metrics[name]; ok {
			boundary := (1 - tolerance) * float64(data.period)
			nextCollection := data.lastCollected.Add(time.Duration(boundary))
			if now.After(nextCollection) {
				data.lastCollected = now
				doCollect = true
			}
		} else {
			matcher.metrics[name] = &CollectData{
				lastCollected: now, // but don't collect the first time
				period:        matcher.matchPeriod(name),
			}
		}

		return doCollect
	}
}

func (matcher *PeriodMatcher) matchPeriod(name string) time.Duration {
	matcher.m.Lock()
	defer matcher.m.Unlock()

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

func (matcher *PeriodMatcher) ConsumeSchedules(sched []*pb.MetricConfigResponse_Schedule) {
	matcher.m.Lock()
	defer matcher.m.Unlock()

	matcher.sched = sched
	matcher.metrics = make(map[string]*CollectData)
}

// TODO: compute GCD for divisibility issues
func (matcher *PeriodMatcher) GetMinPeriod() time.Duration {
	if len(matcher.sched) == 0 {
		panic("matcher has not consumed any schedules")
	}

	minPeriod := matcher.sched[0].PeriodSec
	for _, schedule := range matcher.sched[1:] {
		if minPeriod > schedule.PeriodSec {
			minPeriod = schedule.PeriodSec
		}
	}

	return time.Duration(minPeriod) * time.Second
}

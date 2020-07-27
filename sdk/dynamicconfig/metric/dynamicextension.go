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

package metric

import (
	"sync"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
)

// This file contains extraneous functionality needed to allow per-metric configuration
// for the Accumulator.

// Extension to accumulator which allows per-metric collection.
type DynamicExtension struct {
	lock sync.Mutex

	// List of current schedules.
	schedules []*pb.MetricConfigResponse_Schedule

	// Maps the instrument to the most frequent period of the schedules it matches.
	// Updated when new config is applied and new instruments are added.
	instrumentPeriod map[string]int32
}

func NewDynamicExtension() *DynamicExtension {
	return &DynamicExtension{
		instrumentPeriod: make(map[string]int32),
	}
}

// Find period associated with the instrument name. If it is cached in
// ext.instrumentPeriod, use that. Otherwise, find the period from the
// current list of schedules (choosing the most frequent if multiple
// schedules match.
func (ext *DynamicExtension) FindPeriod(name string) int32 {
	ext.lock.Lock()
	defer ext.lock.Unlock()

	// Check if period associated with instrument name is cached. If so return it.
	if period, ok := ext.instrumentPeriod[name]; ok {
		return period
	}

	// Find schedules that matches with instrument name, and return the most
	// frequent associated period.
	var minPeriod int32 = 0
	for _, schedule := range ext.schedules {
		// To match, name must match at least one InclusionPattern and no
		// ExclusionPatterns.
		if metricpattern.Matches(name, schedule.InclusionPatterns) &&
			!metricpattern.Matches(name, schedule.ExclusionPatterns) &&
			// Check if the period is the smallest of all those from
			// matching schedules so far.
			(minPeriod == 0 || minPeriod > schedule.PeriodSec) {
			minPeriod = schedule.PeriodSec
		}
	}

	ext.instrumentPeriod[name] = minPeriod
	return minPeriod
}

// Clear instrumentPeriod cache.
func (ext *DynamicExtension) Clear() {
	ext.lock.Lock()
	defer ext.lock.Unlock()
	ext.instrumentPeriod = make(map[string]int32)
}

func (ext *DynamicExtension) GetSchedules() []*pb.MetricConfigResponse_Schedule {
	ext.lock.Lock()
	defer ext.lock.Unlock()
	return ext.schedules
}

func (ext *DynamicExtension) SetSchedules(schedules []*pb.MetricConfigResponse_Schedule) {
	ext.lock.Lock()
	defer ext.lock.Unlock()
	ext.schedules = schedules
}

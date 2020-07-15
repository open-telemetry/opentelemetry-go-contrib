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

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
)

// This file contains extraneous functionality needed to allow per-metric configuration
// for the Accumulator.

// Extension to accumulator which allows per-metric collection.
type DynamicExtension struct {
	lock sync.Mutex

	// List of current schedules.
	schedules []*pb.ConfigResponse_MetricConfig_Schedule

	// Maps the instrument to most frequent CollectionPeriod of the schedules it matches.
	// Updated when new config is applied and new instruments are added.
	instrumentPeriod map[string]pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod
}

func NewDynamicExtension() *DynamicExtension {
	return &DynamicExtension{
		instrumentPeriod: make(map[string]pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod),
	}
}

// Find period associated with the instrument name. If it is cached in
// ext.instrumentPeriod, use that. Otherwise, find the period from the
// current list of schedules (choosing the most frequent if multiple
// schedules match.
func (ext *DynamicExtension) FindPeriod(name string) pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod {
	ext.lock.Lock()
	defer ext.lock.Unlock()

	// Check if period associated with instrument name is cached. If so return it.
	if period, ok := ext.instrumentPeriod[name]; ok {
		return period
	}

	// Find schedules that matches with instrument name, and return the most
	// frequent associated CollectionPeriod.
	var minPeriod pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod = 0
	for _, schedule := range ext.schedules {
		// To match, name must match at least one InclusionPattern and no
		// ExclusionPatterns.
		if metricpattern.Matches(name, schedule.InclusionPatterns) &&
			!metricpattern.Matches(name, schedule.ExclusionPatterns) &&
			// Check if the CollectionPeriod is the smallest of all those from
			// matching schedules so far.
			(minPeriod == 0 || minPeriod > schedule.Period) {
			minPeriod = schedule.Period
		}
	}

	ext.instrumentPeriod[name] = minPeriod
	return minPeriod
}

// Clear instrumentPeriod cache.
func (ext *DynamicExtension) Clear() {
	ext.lock.Lock()
	defer ext.lock.Unlock()
	ext.instrumentPeriod = make(map[string]pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod)
}

func (ext *DynamicExtension) GetSchedules() []*pb.ConfigResponse_MetricConfig_Schedule {
	ext.lock.Lock()
	defer ext.lock.Unlock()
	return ext.schedules
}

func (ext *DynamicExtension) SetSchedules(schedules []*pb.ConfigResponse_MetricConfig_Schedule) {
	ext.lock.Lock()
	defer ext.lock.Unlock()
	ext.schedules = schedules
}

// CollectOptions contains optional parameters for Accumulator.Collect().
type CollectOptions struct {
	// All instruments with a CollectPeriod included in `periods` should be
	// exported.
	Periods []pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod
}

// CollectOption is the interface that applies the optional parameter.
type CollectOption interface {
	// Apply sets the Option value of a CollectOptions.
	Apply(*CollectOptions)
}

// WithPeriods sets the Periods option of a CollectOptions.
func WithPeriods(
	periods []pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod,
) CollectOption {
	return periodsOption{periods}
}

type periodsOption struct {
	periods []pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod
}

func (o periodsOption) Apply(config *CollectOptions) {
	config.Periods = o.periods
}

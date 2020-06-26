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

	pb "github.com/vmingchen/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
)

// This file contains extraneous functionality needed to allow per-metric configuration
// for the Accumulator.

// Extension to accumulator which allows per-metric collection.
type DynamicExtension struct {
	lock *sync.Mutex

	// List of current schedules.
	schedules *[]*pb.ConfigResponse_MetricConfig_Schedule

	// Maps the instrument to its CollectionPeriod. Updated when new config
	// is applied and new instruments are added.
	instrumentPeriod map[string]pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod
}

func NewDynamicExtension(
	lock *sync.Mutex,
	schedules *[]*pb.ConfigResponse_MetricConfig_Schedule,
	instrumentPeriod map[string]pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod,
) *DynamicExtension {
	return &DynamicExtension{
		lock: lock, 
		schedules: schedules,
		instrumentPeriod: instrumentPeriod,
	}
}

// CollectConfig contains optional parameters for Accumulator.Collect().
type CollectConfig struct {
	// All instruments with a CollectPeriod included in `periods` should be
	// exported.
	Periods []pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod
}

// CollectOption is the interface that applies the optional parameter.
type CollectOption interface {
	// Apply sets the Option value of a CollectConfig.
	Apply(*CollectConfig)
}

// WithPeriods sets the Periods option of a CollectConfig.
func WithPeriods(
	periods []pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod,
) CollectOption {
	return periodsOption{periods}
}

type periodsOption struct {
	periods []pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod
}

func (o periodsOption) Apply(config *CollectConfig) {
	config.Periods = o.periods
}

// Find schedule that matches with instrument name, and return the one with the
// minimal CollectionPeriod.
func FindPeriod(
	name string,
	schedules *[]*pb.ConfigResponse_MetricConfig_Schedule,
) pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod {
	var period pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod = 0

	for _, schedule := range *schedules {
		schedulePeriod := schedule.Period

		// To match, name must match at least one InclusionPattern and no
		// ExclusionPatterns. If it matches multiple schedules, take the one
		// with the smallest CollectionPeriod.
		if metricpattern.Matches(name, schedule.InclusionPatterns) &&
		!metricpattern.Matches(name, schedule.ExclusionPatterns) &&
		(period == 0 || period > schedulePeriod) {
			period = schedulePeriod
		}
	}
	return period
}

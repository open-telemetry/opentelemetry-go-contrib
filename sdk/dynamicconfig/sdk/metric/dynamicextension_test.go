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
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"
)

const InstrumentName string = "One Fish"

var MatchingPatterns1 = []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals{
			Equals: "One Fish",
		},
	},
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals{
			Equals: "Two Fish",
		},
	},
}
var MatchingPatterns2 = []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith{
			StartsWith: "One",
		},
	},
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith{
			StartsWith: "Two",
		},
	},
}
var NotMatchingPatterns1 = []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals{
			Equals: "Red Fish",
		},
	},
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals{
			Equals: "Blue Fish",
		},
	},
}
var NotMatchingPatterns2 = []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith{
			StartsWith: "Red",
		},
	},
	{
		Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith{
			StartsWith: "Blue",
		},
	},
}

// Test that we should get the associated period from instrumentPeriod if it's
// been cached before.
func TestFindPeriodCached(t *testing.T) {
	ext := NewDynamicExtension()

	ext.instrumentPeriod["One Fish"] = 1

	notMatchingSchedule := pb.ConfigResponse_MetricConfig_Schedule{
		InclusionPatterns: MatchingPatterns1,
		Period:            5,
	}
	ext.schedules = []*pb.ConfigResponse_MetricConfig_Schedule{&notMatchingSchedule}

	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_SEC_1, ext.FindPeriod(InstrumentName))
}

func TestFindPeriodMinimum(t *testing.T) {
	ext := NewDynamicExtension()

	matchingSchedule1 := pb.ConfigResponse_MetricConfig_Schedule{
		InclusionPatterns: MatchingPatterns1,
		ExclusionPatterns: NotMatchingPatterns1,
		Period:            5,
	}
	matchingSchedule2 := pb.ConfigResponse_MetricConfig_Schedule{
		InclusionPatterns: MatchingPatterns2,
		ExclusionPatterns: NotMatchingPatterns2,
		Period:            1,
	}
	ext.schedules = []*pb.ConfigResponse_MetricConfig_Schedule{
		&matchingSchedule1,
		&matchingSchedule2,
	}

	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_SEC_1, ext.FindPeriod(InstrumentName))
	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_SEC_1, ext.instrumentPeriod[InstrumentName])
}

func TestFindPeriodExcluded(t *testing.T) {
	ext := NewDynamicExtension()

	matchingSchedule := pb.ConfigResponse_MetricConfig_Schedule{
		InclusionPatterns: MatchingPatterns1,
		ExclusionPatterns: MatchingPatterns2,
		Period:            5,
	}
	ext.schedules = []*pb.ConfigResponse_MetricConfig_Schedule{&matchingSchedule}

	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_NONE, ext.FindPeriod(InstrumentName))
	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_NONE, ext.instrumentPeriod[InstrumentName])
}

func TestFindPeriodRightMatch(t *testing.T) {
	ext := NewDynamicExtension()

	notMatchingSchedule := pb.ConfigResponse_MetricConfig_Schedule{
		InclusionPatterns: NotMatchingPatterns1,
		ExclusionPatterns: MatchingPatterns1,
		Period:            1,
	}
	matchingSchedule := pb.ConfigResponse_MetricConfig_Schedule{
		InclusionPatterns: MatchingPatterns2,
		ExclusionPatterns: NotMatchingPatterns2,
		Period:            5,
	}
	ext.schedules = []*pb.ConfigResponse_MetricConfig_Schedule{
		&notMatchingSchedule,
		&matchingSchedule,
	}

	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_SEC_5, ext.FindPeriod(InstrumentName))
	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_SEC_5, ext.instrumentPeriod[InstrumentName])
}

func TestFindPeriodNoSchedules(t *testing.T) {
	ext := NewDynamicExtension()

	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_NONE, ext.FindPeriod(InstrumentName))
	require.Equal(t, pb.ConfigResponse_MetricConfig_Schedule_NONE, ext.instrumentPeriod[InstrumentName])
}

func TestClear(t *testing.T) {
	ext := NewDynamicExtension()

	ext.instrumentPeriod["One Fish"] = 1

	ext.Clear()

	require.Equal(t, 0, len(ext.instrumentPeriod))
}

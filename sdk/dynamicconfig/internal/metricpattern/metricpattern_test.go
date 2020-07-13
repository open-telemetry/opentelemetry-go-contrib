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

package metricpattern_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
)

const InstrumentName string = "One Fish"
var EqualsPattern = pb.ConfigResponse_MetricConfig_Schedule_Pattern{
	Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals {
		Equals: "One Fish",
	},
}
var StartsPattern = pb.ConfigResponse_MetricConfig_Schedule_Pattern {
	Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith {
		StartsWith: "One",
	},
}
var MismatchPattern = pb.ConfigResponse_MetricConfig_Schedule_Pattern {
	Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals {
		Equals: "One Whale",
	},
}
var NotEqualsPattern = pb.ConfigResponse_MetricConfig_Schedule_Pattern {
	Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals {
		Equals: "Blue Whale",
	},
}
var NotStartsPattern = pb.ConfigResponse_MetricConfig_Schedule_Pattern {
	Match: &pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith {
		StartsWith: "Two",
	},
}

func TestMatchEquals(t *testing.T) {
	patterns := []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{&EqualsPattern}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestMatchStartsWith(t *testing.T) {
	patterns := []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{&StartsPattern}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestMatchMismatch(t *testing.T) {
	patterns := []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{&MismatchPattern}

	require.False(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestOneMatch(t *testing.T) {
	patterns := []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
		&EqualsPattern,
		&MismatchPattern,
	}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestMultipleMatch(t *testing.T) {
	patterns := []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
		&EqualsPattern,
		&StartsPattern,
		&MismatchPattern,
	}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestNoMatch(t *testing.T) {
	patterns := []*pb.ConfigResponse_MetricConfig_Schedule_Pattern{
		&NotEqualsPattern,
		&NotStartsPattern,
	}

	require.False(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestEmptyPatterns(t *testing.T) {
	patterns := []
	require.False(t, metricpattern.Matches(InstrumentName, patterns))
}

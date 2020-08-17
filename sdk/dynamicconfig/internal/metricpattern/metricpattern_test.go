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

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/metricpattern"
	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
)

const InstrumentName string = "One Fish"

var EqualsPattern = pb.MetricConfigResponse_Schedule_Pattern{
	Match: &pb.MetricConfigResponse_Schedule_Pattern_Equals{
		Equals: "One Fish",
	},
}
var StartsPattern = pb.MetricConfigResponse_Schedule_Pattern{
	Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
		StartsWith: "One",
	},
}
var MismatchPattern = pb.MetricConfigResponse_Schedule_Pattern{
	Match: &pb.MetricConfigResponse_Schedule_Pattern_Equals{
		Equals: "One Whale",
	},
}
var NotEqualsPattern = pb.MetricConfigResponse_Schedule_Pattern{
	Match: &pb.MetricConfigResponse_Schedule_Pattern_Equals{
		Equals: "Blue Whale",
	},
}
var NotStartsPattern = pb.MetricConfigResponse_Schedule_Pattern{
	Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
		StartsWith: "Two",
	},
}
var WildcardStartsPattern = pb.MetricConfigResponse_Schedule_Pattern{
	Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
		StartsWith: "*",
	},
}

func TestMatchEquals(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{&EqualsPattern}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestMatchStartsWith(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{&StartsPattern}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestMatchMismatch(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{&MismatchPattern}

	require.False(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestOneMatch(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{
		&EqualsPattern,
		&MismatchPattern,
	}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestMultipleMatch(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{
		&EqualsPattern,
		&StartsPattern,
		&MismatchPattern,
	}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestNoMatch(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{
		&NotEqualsPattern,
		&NotStartsPattern,
	}

	require.False(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestEmptyPatterns(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{}
	require.False(t, metricpattern.Matches(InstrumentName, patterns))
}

func TestWildcardPatterns(t *testing.T) {
	patterns := []*pb.MetricConfigResponse_Schedule_Pattern{
		&WildcardStartsPattern,
	}

	require.True(t, metricpattern.Matches(InstrumentName, patterns))
}

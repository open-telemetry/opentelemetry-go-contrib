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

package notify

import (
	"testing"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"github.com/stretchr/testify/require"
)

func GetDefaultConfig(period int32, fingerprint []byte) *MetricConfig {
	pattern := pb.MetricConfigResponse_Schedule_Pattern{
		Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{StartsWith: "*"},
	}
	schedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{&pattern},
		PeriodSec:         period,
	}

	return &MetricConfig{
		pb.MetricConfigResponse{
			Fingerprint: fingerprint,
			Schedules:   []*pb.MetricConfigResponse_Schedule{&schedule},
		},
	}
}

func TestMetricConfigValidate(t *testing.T) {
	schedule1 := pb.MetricConfigResponse_Schedule{PeriodSec: -1}
	schedule2 := pb.MetricConfigResponse_Schedule{PeriodSec: 1}

	config := &MetricConfig{
		pb.MetricConfigResponse{
			Schedules: []*pb.MetricConfigResponse_Schedule{&schedule1},
		},
	}
	require.NotNil(t, config.Validate())

	config = &MetricConfig{
		pb.MetricConfigResponse{
			Schedules: []*pb.MetricConfigResponse_Schedule{&schedule2},
		},
	}
	require.Equal(t, nil, config.Validate())
}

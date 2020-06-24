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

package dynamicconfig_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/vmingchen/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"

	"go.opentelemetry.io/contrib/exporters/metric/dynamicconfig"
)

func TestEquals(t *testing.T) {
	fooConfig1 := dynamicconfig.GetDefaultConfig(1, []byte{'f', 'o', 'o'})
	fooConfig2 := dynamicconfig.GetDefaultConfig(2, []byte{'f', 'o', 'o'})
	barConfig := dynamicconfig.GetDefaultConfig(1, []byte{'b', 'a', 'r'})

	require.True(t, fooConfig1.Equals(fooConfig2))
	require.False(t, fooConfig1.Equals(barConfig))
}

func TestValidate(t *testing.T) {
	schedule1 := pb.ConfigResponse_MetricConfig_Schedule{Period: 0}
	schedule2 := pb.ConfigResponse_MetricConfig_Schedule{Period: 1}

	config := &dynamicconfig.Config{
		pb.ConfigResponse{
			MetricConfig: &pb.ConfigResponse_MetricConfig{
				Schedules: []*pb.ConfigResponse_MetricConfig_Schedule{&schedule2, &schedule2},
			},
		},
	}
	require.Equal(t, config.Validate(), errors.New("Config must have exactly one Schedule"))

	config = &dynamicconfig.Config{
		pb.ConfigResponse{
			MetricConfig: &pb.ConfigResponse_MetricConfig{
				Schedules: []*pb.ConfigResponse_MetricConfig_Schedule{&schedule1},
			},
		},
	}
	require.Equal(t, config.Validate(), errors.New("Period must be positive"))

	config = &dynamicconfig.Config{
		pb.ConfigResponse{
			MetricConfig: &pb.ConfigResponse_MetricConfig{
				Schedules: []*pb.ConfigResponse_MetricConfig_Schedule{&schedule2},
			},
		},
	}
	require.Equal(t, config.Validate(), nil)
}

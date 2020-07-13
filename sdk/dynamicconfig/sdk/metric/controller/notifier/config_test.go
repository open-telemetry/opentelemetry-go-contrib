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

package notifier_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"

	notify "go.opentelemetry.io/contrib/sdk/dynamicconfig/sdk/metric/controller/notifier"
)

func TestEquals(t *testing.T) {
	fooConfig1 := notify.GetDefaultConfig(1, []byte{'f', 'o', 'o'})
	fooConfig2 := notify.GetDefaultConfig(2, []byte{'f', 'o', 'o'})
	barConfig := notify.GetDefaultConfig(1, []byte{'b', 'a', 'r'})

	require.True(t, fooConfig1.Equals(fooConfig2))
	require.False(t, fooConfig1.Equals(barConfig))
}

func TestMetricConfigValidate(t *testing.T) {
	schedule1 := pb.ConfigResponse_MetricConfig_Schedule{Period: -1}
	schedule2 := pb.ConfigResponse_MetricConfig_Schedule{Period: 1}

	config := &notify.Config{
		pb.ConfigResponse{},
	}
	require.Equal(t, errors.New("No MetricConfig"), config.ValidateMetricConfig())

	config = &notify.Config{
		pb.ConfigResponse{
			MetricConfig: &pb.ConfigResponse_MetricConfig{
				Schedules: []*pb.ConfigResponse_MetricConfig_Schedule{&schedule1},
			},
		},
	}
	require.Equal(t, errors.New("Periods must be positive"), config.ValidateMetricConfig())

	config = &notify.Config{
		pb.ConfigResponse{
			MetricConfig: &pb.ConfigResponse_MetricConfig{
				Schedules: []*pb.ConfigResponse_MetricConfig_Schedule{&schedule2},
			},
		},
	}
	require.Equal(t, nil, config.ValidateMetricConfig())
}

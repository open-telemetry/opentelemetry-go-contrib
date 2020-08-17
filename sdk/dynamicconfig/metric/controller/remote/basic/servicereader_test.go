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
package basic

import (
	"testing"

	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
)

func TestNewServiceReader(t *testing.T) {
	server := MockServer{}
	stop, addr := server.Run(t)
	defer stop()

	reader, err := NewServiceReader(addr, nil)
	if err != nil {
		t.Errorf("fail to start service reader: %v", err)
	}

	if err := reader.Stop(); err != nil {
		t.Errorf("fail to stop service reader: %v", err)
	}
}

func TestReadConfig(t *testing.T) {
	config := &pb.MetricConfigResponse{
		Schedules: []*pb.MetricConfigResponse_Schedule{
			{
				InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
					{
						Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
							StartsWith: "*",
						},
					},
				},
				PeriodSec: 5,
			},
		},
	}

	server := MockServer{Config: config}
	stop, addr := server.Run(t)
	defer stop()

	reader, err := NewServiceReader(addr, nil)
	if err != nil {
		t.Errorf("fail to start service reader: %v", err)
	}
	defer func() {
		if err := reader.Stop(); err != nil {
			t.Errorf("fail to stop reader: %v", err)
		}
	}()

	resp, err := reader.ReadConfig()
	if err != nil {
		t.Errorf("fail to read config: %v", err)
	}

	if resp.Schedules[0].PeriodSec != config.Schedules[0].PeriodSec {
		t.Errorf("schedules do not match, resp: %v", resp)
	}
}

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
	"time"

	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/remote"
	controllerTest "go.opentelemetry.io/otel/sdk/metric/controller/test"
)

func TestMonitorChanges(t *testing.T) {
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

	mockClock := controllerTest.NewMockClock()
	monitor := NewMonitor(addr, nil)
	monitor.clock = mockClock

	mch := remote.NewMonitorChannel()
	monitor.MonitorChanges(mch)

	select {
	case scheds := <-mch.Data:
		if scheds[0].PeriodSec != config.Schedules[0].PeriodSec {
			t.Errorf("config does not match received data: %v", scheds)
		}
	case err := <-mch.Err:
		t.Errorf("monitor failed: %v", err)
	}

	config.Schedules[0].PeriodSec = 10
	config.SuggestedWaitTimeSec = 5
	mockClock.Add(DefaultCheckFrequency)

	select {
	case scheds := <-mch.Data:
		if scheds[0].PeriodSec != config.Schedules[0].PeriodSec {
			t.Errorf("config does not match received data: %v", scheds)
		}
	case err := <-mch.Err:
		t.Errorf("monitor failed: %v", err)
	}

	config.Schedules[0].PeriodSec = 15
	mockClock.Add(5 * time.Second)

	select {
	case scheds := <-mch.Data:
		if scheds[0].PeriodSec != config.Schedules[0].PeriodSec {
			t.Errorf("config does not match received data: %v", scheds)
		}
	case err := <-mch.Err:
		t.Errorf("monitor failed: %v", err)
	}
}

func TestUpdateWaitTime(t *testing.T) {
	monitor := NewMonitor("", nil)
	mockClock := controllerTest.NewMockClock()
	monitor.clock = mockClock
	monitor.ticker = monitor.clock.Ticker(1 * time.Second)

	monitor.updateWaitTime(10)
	mockClock.Add(1 * time.Second)

	select {
	case <-monitor.ticker.C():
		t.Errorf("clock ticked after 1 second, not 10")
	default:
	}

	mockClock.Add(9 * time.Second)

	select {
	case <-monitor.ticker.C():
	default:
		t.Errorf("clock should have ticked by now, after 10 seconds")
	}

	monitor.updateWaitTime(15)
	mockClock.Add(10 * time.Second)

	select {
	case <-monitor.ticker.C():
		t.Errorf("clock ticked after 10 seconds, not 15")
	default:
	}

	mockClock.Add(5 * time.Second)

	select {
	case <-monitor.ticker.C():
	default:
		t.Errorf("clock should have ticked by now, after 15 seconds")
	}

	monitor.updateWaitTime(0)
	mockClock.Add(15 * time.Second)

	select {
	case <-monitor.ticker.C():
	default:
		t.Errorf("clock should have ticked by now, after 15 seconds")
	}
}

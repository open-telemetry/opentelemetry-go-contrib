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

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify"
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
	notifier := NewNotifier(addr, nil)
	notifier.clock = mockClock

	mch := notify.NewMonitorChannel()
	go notifier.MonitorChanges(mch)

	select {
	case data := <-mch.Data:
		if data.Schedules[0].PeriodSec != config.Schedules[0].PeriodSec {
			t.Errorf("config does not match received data: %v", data)
		}
	case err := <-mch.Err:
		t.Errorf("monitor failed: %v", err)
	}

	config.Schedules[0].PeriodSec = 10
	config.SuggestedWaitTimeSec = 5
	mockClock.Add(DefaultCheckFrequency)

	select {
	case data := <-mch.Data:
		if data.Schedules[0].PeriodSec != config.Schedules[0].PeriodSec {
			t.Errorf("config does not match received data: %v", data)
		}
	case err := <-mch.Err:
		t.Errorf("monitor failed: %v", err)
	}

	config.Schedules[0].PeriodSec = 15
	mockClock.Add(5 * time.Second)

	select {
	case data := <-mch.Data:
		if data.Schedules[0].PeriodSec != config.Schedules[0].PeriodSec {
			t.Errorf("config does not match received data: %v", data)
		}
	case err := <-mch.Err:
		t.Errorf("monitor failed: %v", err)
	}
}

func TestUpdateWaitTime(t *testing.T) {
	notifier := NewNotifier("", nil)
	mockClock := controllerTest.NewMockClock()
	notifier.clock = mockClock
	notifier.ticker = notifier.clock.Ticker(1 * time.Second)

	notifier.updateWaitTime(10)
	mockClock.Add(1 * time.Second)

	select {
	case <-notifier.ticker.C():
		t.Errorf("clock ticked after 1 second, not 10")
	default:
	}

	mockClock.Add(9 * time.Second)

	select {
	case <-notifier.ticker.C():
	default:
		t.Errorf("clock should have ticked by now, after 10 seconds")
	}

	notifier.updateWaitTime(15)
	mockClock.Add(10 * time.Second)

	select {
	case <-notifier.ticker.C():
		t.Errorf("clock ticked after 10 seconds, not 15")
	default:
	}

	mockClock.Add(5 * time.Second)

	select {
	case <-notifier.ticker.C():
	default:
		t.Errorf("clock should have ticked by now, after 15 seconds")
	}

	notifier.updateWaitTime(0)
	mockClock.Add(15 * time.Second)

	select {
	case <-notifier.ticker.C():
	default:
		t.Errorf("clock should have ticked by now, after 15 seconds")
	}
}

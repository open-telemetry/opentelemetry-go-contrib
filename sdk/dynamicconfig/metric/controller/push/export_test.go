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

package push

import (
	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify/mock"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
)

func (c *Controller) SetClock(clock controllerTime.Clock) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.clock = clock
}

func (c *Controller) SetNotifier(notifier notify.Notifier) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.notifier = notifier
}

func (c *Controller) SetPeriod(period int32) {
	config := pb.MetricConfigResponse{
		Schedules: []*pb.MetricConfigResponse_Schedule{
			{
				InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
					{
						Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
							StartsWith: "",
						},
					},
				},
				PeriodSec: period,
			},
		},
	}

	notifier := mock.NewNotifier()
	notifier.Receive(&notify.MetricConfig{config})
	c.SetNotifier(notifier)
}

func (c *Controller) SetDone() {
	c.done = make(chan struct{})
}

func (c *Controller) WaitDone() {
	<-c.done
}

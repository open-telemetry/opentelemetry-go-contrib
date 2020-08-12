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

package mock

import (
	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/remote"
)

type Monitor struct {
	data chan []*pb.MetricConfigResponse_Schedule
}

func NewMonitor() *Monitor {
	return &Monitor{make(chan []*pb.MetricConfigResponse_Schedule)}
}

func (m *Monitor) Receive(scheds []*pb.MetricConfigResponse_Schedule) {
	go func() { m.data <- scheds }()
}

// TODO: switch to sending data paradigm
func (m *Monitor) MonitorChanges(mch remote.MonitorChannel) {
	go func() {
		for {
			select {
			case config := <-m.data:
				mch.Data <- config
			case <-mch.Quit:
				return
			}
		}
	}()
}

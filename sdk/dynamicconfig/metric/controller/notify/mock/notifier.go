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
	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify"
)

type Notifier struct {
	data chan []*pb.MetricConfigResponse_Schedule
}

func NewNotifier() *Notifier {
	return &Notifier{make(chan []*pb.MetricConfigResponse_Schedule)}
}

func (n *Notifier) Receive(scheds []*pb.MetricConfigResponse_Schedule) {
	go func() { n.data <- scheds }()
}

// TODO: switch to sending data paradigm
func (n *Notifier) MonitorChanges(mch notify.MonitorChannel) {
	for {
		select {
		case config := <-n.data:
			mch.Data <- config
		case <-mch.Quit:
			return
		}
	}
}

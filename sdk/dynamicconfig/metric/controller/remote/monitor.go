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

// Package remote provides utilities that controllers may use to communicate
// with an upstream dynamic configuration service.
package remote

import pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"

// A Monitor is an entity that watches the upstream service for updates, and
// communicates those updates through a MonitorChannel.
type Monitor interface {
	// MonitorChanges launches a goroutine that monitors the configuration
	// service for updates.
	MonitorChanges(mch MonitorChannel)
}

// A MonitorChannel holds the communication channels that a Monitor uses to
// inform a controller of updates from an upstream service.
type MonitorChannel struct {
	// Data contains updated metric schedules
	Data chan []*pb.MetricConfigResponse_Schedule

	// Err reports any errors in the system
	Err chan error

	// Quit is used by the controller to shut down the MonitorChannel.
	Quit chan struct{}
}

// NewMonitorChannel instantiates a MonitorChannel and its associated channels.
func NewMonitorChannel() MonitorChannel {
	return MonitorChannel{
		Data: make(chan []*pb.MetricConfigResponse_Schedule),
		Err:  make(chan error),
		Quit: make(chan struct{}),
	}
}

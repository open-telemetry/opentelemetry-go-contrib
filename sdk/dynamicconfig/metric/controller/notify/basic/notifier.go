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
	"time"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/transform"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
	"go.opentelemetry.io/otel/sdk/resource"
)

const DefaultCheckFrequency = 30 * time.Minute

// A Notifier monitors a config service for a config changing, then letting
// all its subscribers know if the config has changed.
//
// All fields except for subscribed and config, which are protected by lock,
// should be read-only once set.
type Notifier struct {
	// How often we check to see if the config service has changed.
	lastWaitTime int32

	// Added for testing time-related functionality.
	clock controllerTime.Clock

	// Optional field for the address of the config service host if the config is
	// non-dynamic.
	configHost string

	// Label to associate configs to individual instances.
	// Optional if config is non-dynamic, mandatory to read from config service.
	resource *resource.Resource

	// Controls when we check the config service for a potential new config. It is
	// set to nil if configHost is empty.
	ticker controllerTime.Ticker
}

// Constructor for a Notifier
func NewNotifier(configHost string, resource *resource.Resource) *Notifier {
	notifier := &Notifier{
		clock:      controllerTime.RealClock{},
		configHost: configHost,
		resource:   resource,
	}

	return notifier
}

// TODO: move to export?
func (n *Notifier) SetClock(clock controllerTime.Clock) {
	n.clock = clock
}

func (n *Notifier) MonitorChanges(mch notify.MonitorChannel) {
	n.ticker = n.clock.Ticker(DefaultCheckFrequency)
	serviceReader, err := NewServiceReader(n.configHost, transform.Resource(n.resource))
	if err != nil {
		mch.Err <- err
	}

	n.tick(mch.Data, mch.Err, serviceReader)
	for {
		select {
		case <-n.ticker.C():
			n.tick(mch.Data, mch.Err, serviceReader)

		case <-mch.Quit:
			n.ticker.Stop()
			if err := serviceReader.Stop(); err != nil {
				mch.Err <- err
			}
			return
		}
	}
}

func (n *Notifier) tick(data chan<- *notify.MetricConfig, errCh chan<- error, serviceReader *ServiceReader) {
	newConfig, err := serviceReader.ReadConfig()
	if err != nil {
		errCh <- err
	}

	if newConfig != nil {
		data <- newConfig
		n.updateWaitTime(newConfig.SuggestedWaitTimeSec)
	}
}

func (n *Notifier) updateWaitTime(waitTime int32) {
	if waitTime > 0 && n.lastWaitTime != waitTime {
		n.ticker.Stop()
		n.lastWaitTime = waitTime
		n.ticker = n.clock.Ticker(time.Duration(n.lastWaitTime))
	}
}

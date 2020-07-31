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
	"time"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/transform"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
	"go.opentelemetry.io/otel/sdk/resource"
)

// A Notifier monitors a config service for a config changing, then letting
// all its subscribers know if the config has changed.
//
// All fields except for subscribed and config, which are protected by lock,
// should be read-only once set.
type Notifier struct {
	// How often we check to see if the config service has changed.
	checkFrequency time.Duration

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
		configHost: configHostOption,
		resource:   resource,
	}

	return notifier, nil
}

// SetClock supports setting a mock clock for testing.  This must be
// called before Start().
func (n *Notifier) SetClock(clock controllerTime.Clock) {
	n.clock = clock
}

type MonitorChannel struct {
	Data <-chan *MetricConfig
	Err  <-chan error
	Quit chan<- struct{}
}

func (n *Notifier) MonitorChanges(mch MonitorChannel) {
	n.ticker = n.clock.Ticker(DefaultCheckFrequency)
	serviceReader, err := NewServiceReader(n.configHost, transform.Resource(n.resource))
	if err != nil {
		mch.Err <- err
	}

	n.tick(mch.Data, mch.Err)
	for {
		select {
		case <-n.ticker.C():
			n.tick(mch.Data, mch.Err)

		case <-mch.Quit:
			n.ticker.Stop()
			if err := serviceReader.Stop(); err != nil {
				errCh <- err
			}
			return
		}
	}
}

func (n *Notifier) tick(data <-chan *MetricConfig, errCh <-chan error) {
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
	if waitTime > 0 && n.checkFrequency != waitTime {
		n.ticker.Stop()
		n.checkFrequency = waitTime
		n.ticker = n.clock.Ticker(n.checkFrequency)
	}
}

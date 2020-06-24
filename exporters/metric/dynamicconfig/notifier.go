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

package dynamicconfig

import (
	"errors"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/internal/transform"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
	"go.opentelemetry.io/otel/sdk/resource"
)

// DefaultCheckFrequency is the default frequency at which we check for new configs.
const DefaultCheckFrequency = time.Minute

type Watcher interface {
	// NOTE: A lock will be held during the execution of both these functions.
	// Please ensure their implementation is not too slow so as to avoid lock-
	// starvation.
	//
	// There is a common lock shared by both functions, ensuring there is no
	// concurrent invocation of these two functions; therefore caller does not
	// need a lock protecting access to members of MetricConfig.
	OnInitialConfig(config *Config) error
	OnUpdatedConfig(config *Config) error
}

// A Notifier monitors a config service for a config changing, then letting
// all its subscribers know if the config has changed.
//
// All fields except for subscribed and config, which are protected by lock,
// should be read-only once set.
type Notifier struct {
	// Used to shut down the config checking routine when we stop Notifier.
	ch chan struct{}

	// How often we check to see if the config service has changed.
	checkFrequency time.Duration

	// Added for testing time-related functionality.
	clock controllerTime.Clock

	// The current config we use as a default if we cannot read from the remote
	// configuration service.
	config *Config

	// Optional field for the address of the config service host if the config is
	// non-dynamic.
	configHost string

	// This protects the config and subscribed fields. Other fields should be
	// read-only once set.
	lock sync.Mutex

	// Label to associate configs to individual instances.
	// Optional if config is non-dynamic, mandatory to read from config service.
	resource *resource.Resource

	// Set of all the notifier's subscribers.
	subscribed map[Watcher]bool

	// Controls when we check the config service for a potential new config. It is
	// set to nil if configHost is empty.
	ticker controllerTime.Ticker

	// This is used to wait for the config checking routine to return when we stop
	// the notifier.
	wg sync.WaitGroup
}

// Constructor for a Notifier
func NewNotifier(defaultConfig *Config, opts ...Option) (*Notifier, error) {
	notifier := &Notifier{
		ch:             make(chan struct{}),
		checkFrequency: DefaultCheckFrequency,
		clock:          controllerTime.RealClock{},
		config:         defaultConfig,
		subscribed:     make(map[Watcher]bool),
	}

	for _, opt := range opts {
		opt.Apply(notifier)
	}

	if notifier.configHost != "" && notifier.resource == nil {
		return nil, errors.New("Missing Resource: required for reading from config service")
	}

	return notifier, nil
}

// SetClock supports setting a mock clock for testing.  This must be
// called before Start().
func (n *Notifier) SetClock(clock controllerTime.Clock) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.clock = clock
}

func (n *Notifier) Start() {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.configHost == "" {
		return
	}

	if n.ticker != nil {
		return
	}

	n.ticker = n.clock.Ticker(n.checkFrequency)
	n.wg.Add(1)
	go n.checkChanges(n.ch)
}

func (n *Notifier) Stop() {
	n.lock.Lock()

	if n.configHost == "" {
		return
	}

	if n.ch == nil {
		return
	}
	close(n.ch)
	n.ch = nil

	n.lock.Unlock()

	n.wg.Wait()
	n.ticker.Stop()
}

func (n *Notifier) Register(watcher Watcher) {
	n.lock.Lock()
	n.subscribed[watcher] = true
	err := watcher.OnInitialConfig(n.config)
	n.lock.Unlock()

	if err != nil {
		log.Printf("Failed to apply config: %v\n", err)
	}
}

func (n *Notifier) Unregister(watcher Watcher) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.subscribed, watcher)
}

func (n *Notifier) checkChanges(ch chan struct{}) {
	serviceReader := NewServiceReader(n.configHost, transform.Resource(n.resource))

	for {
		select {
		case <-ch:
			n.wg.Done()
			return
		case <-n.ticker.C():
			newConfig, err := serviceReader.readConfig()
			if err != nil {
				log.Printf("Failed to read from config service: %v\n", err)
				break
			}

			n.lock.Lock()
			if !n.config.Equals(newConfig) {
				n.config = newConfig

				for watcher := range n.subscribed {
					err = watcher.OnUpdatedConfig(newConfig)
					if err != nil {
						log.Printf("Failed to apply config: %v\n", err)
						break
					}
				}
			}
			n.lock.Unlock()
		}
	}
}

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

package push // import "go.opentelemetry.io/contrib/exporters/metric/dynamicconfig/push"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/dynamicconfig"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/metric/registry"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
)

// DefaultPushPeriod is the default time interval between pushes.
const DefaultPushPeriod = 10 * time.Second

// Controller organizes a periodic push of metric data.
type Controller struct {
	lock        sync.Mutex
	accumulator *sdk.Accumulator
	provider    *registry.Provider
	processor   *basic.Processor
	exporter    export.Exporter
	wg          sync.WaitGroup
	ch          chan bool
	period      time.Duration
	timeout     time.Duration
	clock       controllerTime.Clock
	ticker      controllerTime.Ticker
	notifier    *dynamicconfig.Notifier
}

// New constructs a Controller, an implementation of metric.Provider,
// using the provided exporter and options to configure an SDK with
// periodic collection.
func New(selector export.AggregatorSelector, exporter export.Exporter, opts ...Option) *Controller {
	c := &Config{
		Period: DefaultPushPeriod,
	}
	for _, opt := range opts {
		opt.Apply(c)
	}
	if c.Timeout == 0 {
		c.Timeout = c.Period
	}

	processor := basic.New(selector, exporter)
	impl := sdk.NewAccumulator(
		processor,
		sdk.WithResource(c.Resource),
	)
	return &Controller{
		provider:    registry.NewProvider(impl),
		accumulator: impl,
		processor:   processor,
		exporter:    exporter,
		ch:          make(chan bool),
		period:      c.Period,
		timeout:     c.Timeout,
		clock:       controllerTime.RealClock{},
		notifier:    c.Notifier,
	}
}

// SetClock supports setting a mock clock for testing.  This must be
// called before Start().
func (c *Controller) SetClock(clock controllerTime.Clock) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.clock = clock
}

// Provider returns a metric.Provider instance for this controller.
func (c *Controller) Provider() metric.Provider {
	return c.provider
}

// Start begins a ticker that periodically collects and exports
// metrics with the configured interval.
func (c *Controller) Start() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.ticker != nil {
		return
	}

	if c.notifier != nil {
		c.notifier.Register(c)
	}

	c.ticker = c.clock.Ticker(c.period)
	c.wg.Add(1)
	go c.run(c.ch)
}

// Stop waits for the background goroutine to return and then collects
// and exports metrics one last time before returning.
func (c *Controller) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.ch == nil {
		return
	}

	close(c.ch)
	c.ch = nil
	c.wg.Wait()
	c.ticker.Stop()
	c.ticker = nil

	c.tick()

	if c.notifier != nil {
		c.notifier.Unregister(c)
	}
}

// We assume that the only metric schedule is one with an inclusion pattern
// that includes all metrics
func (c *Controller) OnInitialConfig(config *dynamicconfig.Config) error {
	err := config.Validate()
	if err != nil {
		return err
	}

	c.period = time.Duration(config.MetricConfig.Schedules[0].Period) * time.Second

	return nil
}

func (c *Controller) OnUpdatedConfig(config *dynamicconfig.Config) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	err := config.Validate()
	if err != nil {
		return err
	}

	// Stop the existing ticker
	// Make a new ticker with the new sampling period
	c.ticker.Stop()
	if len(config.MetricConfig.Schedules) > 0 {
		c.period = time.Duration(config.MetricConfig.Schedules[0].Period) * time.Second
	}
	c.ticker = c.clock.Ticker(c.period)

	// Let the controller know to check the new ticker
	c.ch <- true

	return nil
}

func (c *Controller) run(ch chan bool) {
	for {
		select {
		// If signal receives 'true', break to check the new ticker
		// If signal receives 'false', that means controller is stopping
		case signal := <-ch:
			if signal {
				break
			}
			c.wg.Done()
			return
		case <-c.ticker.C():
			c.tick()
		}
	}
}

func (c *Controller) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	c.processor.Lock()
	defer c.processor.Unlock()

	c.processor.StartCollection()
	c.accumulator.Collect(ctx)
	if err := c.processor.FinishCollection(); err != nil {
		global.Handle(err)
	}

	if err := c.exporter.Export(ctx, c.processor.CheckpointSet()); err != nil {
		global.Handle(err)
	}
}

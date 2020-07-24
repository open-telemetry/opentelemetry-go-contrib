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

// import "go.opentelemetry.io/contrib/sdk/dynamicconfig/sdk/metric/controller/push"
package push

import (
	"context"
	"sync"
	"time"

	sdk "go.opentelemetry.io/contrib/sdk/dynamicconfig/sdk/metric"
	notify "go.opentelemetry.io/contrib/sdk/dynamicconfig/sdk/metric/controller/notifier"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/metric/registry"
	export "go.opentelemetry.io/otel/sdk/export/metric"
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
	notifier    *notify.Notifier
	// Used to store and apply metric schedules
	dynamicExtension *sdk.DynamicExtension
	// Timestamp all metrics with period were last exported
	lastCollected map[int32]time.Time
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

	var extension *sdk.DynamicExtension = nil
	if c.Notifier != nil {
		extension = sdk.NewDynamicExtension()
	}

	processor := basic.New(selector, exporter)
	impl := sdk.NewAccumulator(
		processor,
		sdk.WithResource(c.Resource),
		sdk.WithDynamicExtension(extension),
	)
	return &Controller{
		provider:         registry.NewProvider(impl),
		accumulator:      impl,
		processor:        processor,
		exporter:         exporter,
		ch:               make(chan bool),
		period:           c.Period,
		timeout:          c.Timeout,
		clock:            controllerTime.RealClock{},
		notifier:         c.Notifier,
		dynamicExtension: extension,
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

// Called by notifier if we Register with one.
func (c *Controller) OnInitialConfig(config *notify.MetricConfig) error {
	err := config.ValidateMetricConfig()
	if err != nil {
		return err
	}

	c.dynamicExtension.SetSchedules(config.Schedules)
	c.period = c.updateFromSchedules()

	return nil
}

// Called by notifier if it receives an update.
func (c *Controller) OnUpdatedConfig(config *notify.MetricConfig) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	err := config.ValidateMetricConfig()
	if err != nil {
		return err
	}

	c.dynamicExtension.Clear()

	c.dynamicExtension.SetSchedules(config.Schedules)
	c.period = c.updateFromSchedules()

	// Stop the existing ticker
	c.ticker.Stop()
	// If no schedules, or all schedules have a period of 0, controller never exports.
	if c.period == 0 {
		return nil
	}

	// Make a new ticker with a new sampling period
	c.ticker = c.clock.Ticker(c.period)

	// Let the controller know to check the new ticker
	c.ch <- true

	return nil
}

// TODO: Return time.Second if periods are not mutually divisible
// Iterate through the schedules for two purposes:
//    -Return the minimal non-zero period
//    -Reset lastCollected, set each period's last collection
//    timestamp to now.
func (c *Controller) updateFromSchedules() time.Duration {
	now := c.clock.Now()

	var minPeriod int32 = 0
	c.lastCollected = make(map[int32]time.Time)

	for _, schedule := range c.dynamicExtension.GetSchedules() {
		// If the period is 0, we do not do anything with it
		if schedule.PeriodSec == 0 {
			continue
		}

		if _, ok := c.lastCollected[schedule.PeriodSec]; !ok {
			c.lastCollected[schedule.PeriodSec] = now
		}

		if minPeriod == 0 || minPeriod > schedule.PeriodSec {
			minPeriod = schedule.PeriodSec
		}
	}

	return time.Duration(minPeriod) * time.Second
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

	if c.notifier == nil {
		c.accumulator.Collect(ctx)
	} else {
		// Export all metrics with the same period at the same time.
		overdue := []int32{}
		now := c.clock.Now()
		// Have a tolerance of 10% of the period
		tolerance := c.period / time.Duration(10)

		for period, lastCollect := range c.lastCollected {
			expectedExportTimeWithTolerance := lastCollect.Add(time.Duration(period) * time.Second - tolerance)

			// Check if enough time elapsed since metrics with `period` were
			// last exported, within the tolerance.
			if expectedExportTimeWithTolerance.Before(now) {
				overdue = append(overdue, period)
				c.lastCollected[period] = now
			}
		}

		if len(overdue) > 0 {
			c.accumulator.Collect(ctx, sdk.WithPeriods(overdue))
		}
	}

	if err := c.processor.FinishCollection(); err != nil {
		global.Handle(err)
	}

	if err := c.exporter.Export(ctx, c.processor.CheckpointSet()); err != nil {
		global.Handle(err)
	}
}

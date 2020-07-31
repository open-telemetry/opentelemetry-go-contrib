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
	"context"
	"sync"
	"time"

	sdk "go.opentelemetry.io/contrib/sdk/dynamicconfig/metric"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/metric/registry"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
)

const fallbackPeriod = 10 * time.Second

// Controller organizes a periodic push of metric data.
type Controller struct {
	lock        sync.Mutex
	accumulator *sdk.Accumulator
	provider    *registry.Provider
	processor   *basic.Processor
	exporter    export.Exporter
	quit        chan bool
	timeout     time.Duration
	clock       controllerTime.Clock
	ticker      controllerTime.Ticker
	notifier    *Notifier
	mch         MonitorChannel
	matcher     *PeriodMatcher
}

// New constructs a Controller, an implementation of metric.Provider,
// using the provided exporter and options to configure an SDK with
// periodic collection.
func New(selector export.AggregatorSelector, exporter export.Exporter, configHost string, opts ...Option) *Controller {
	c := &Config{}
	for _, opt := range opts {
		opt.Apply(c)
	}
	if c.Timeout == 0 {
		c.Timeout = fallbackPeriod
	}

	processor := basic.New(selector, exporter)
	impl := sdk.NewAccumulator(
		processor,
		sdk.WithResource(c.Resource),
	)

	notifier := NewNotifier(configHost, c.Resource)
	mch := MonitorChannel{
		Data: make(chan *MetricConfig),
		Err:  make(chan error),
		Quit: make(chan struct{}),
	}

	return &Controller{
		provider:    registry.NewProvider(impl),
		accumulator: impl,
		processor:   processor,
		exporter:    exporter,
		quit:        make(chan bool),
		period:      c.Period,
		timeout:     c.Timeout,
		clock:       controllerTime.RealClock{},
		notifier:    notifier,
		mch:         mch,
		matcher:     &PeriodMatcher{},
	}
}

// SetClock supports setting a mock clock for testing. This must be
// called before Start().
// TODO: use export?
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

	c.ticker = c.clock.Ticker(fallbackPeriod)

	go c.notifier.MonitorChanges(c.mch)
	go c.run()
}

// Stop waits for the background goroutine to return and then collects
// and exports metrics one last time before returning.
func (c *Controller) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.quit == nil {
		return
	}

	close(c.quit)
	c.quit = nil
	c.ticker.Stop()
	c.ticker = nil

	c.tick()
}

func (c *Controller) run() {
	for {
		select {
		case <-c.quit:
			close(c.mch.Quit)
			return
		case <-c.ticker.C():
			c.tick()
		case data := <-c.mch.Data:
			c.update(data)
		case err := <-c.mch.Err:
			global.Handle(err)
		}
	}
}

func (c *Controller) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	c.processor.Lock()
	defer c.processor.Unlock()

	c.processor.StartCollection()
	rule := c.matcher.BuildRule(c.clock.Now())
	c.accumulator.Collect(ctx, rule)

	if err := c.processor.FinishCollection(); err != nil {
		global.Handle(err)
	}

	if err := c.exporter.Export(ctx, c.processor.CheckpointSet()); err != nil {
		global.Handle(err)
	}
}

func (c *Controller) update(data *MetricConfig) {
	c.matcher.ConsumeSchedules(data.Schedules)
	minPeriod := c.matcher.GetMinPeriod()
	if c.period != minPeriod {
		c.ticker.Stop()
		c.ticker = c.clock.Ticker(matcher.GetMinPeriod())
	}
}

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

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify"
	nbasic "go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify/basic"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/metric/registry"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	controllerTime "go.opentelemetry.io/otel/sdk/metric/controller/time"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
)

const fallbackPeriod = 10 * time.Minute

// Controller organizes a periodic push of metric data.
type Controller struct {
	lock        sync.Mutex
	accumulator *sdk.Accumulator
	provider    *registry.Provider
	processor   *basic.Processor
	exporter    export.Exporter
	lastPeriod  time.Duration
	quit        chan struct{}
	done        chan struct{}
	isRunning   bool
	timeout     time.Duration
	clock       controllerTime.Clock
	ticker      controllerTime.Ticker
	notifier    notify.Notifier
	mch         notify.MonitorChannel
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

	notifier := nbasic.NewNotifier(configHost, c.Resource)
	mch := notify.NewMonitorChannel()

	return &Controller{
		provider:    registry.NewProvider(impl),
		accumulator: impl,
		processor:   processor,
		exporter:    exporter,
		quit:        make(chan struct{}),
		timeout:     c.Timeout,
		clock:       controllerTime.RealClock{},
		notifier:    notifier,
		mch:         mch,
		matcher:     &PeriodMatcher{},
	}
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

	if c.isRunning {
		return
	}

	c.isRunning = true
	c.matcher.MarkStart(c.clock.Now())
	c.notifier.MonitorChanges(c.mch)
	go c.run()
}

// Stop waits for the background goroutine to return and then collects
// and exports metrics one last time before returning.
func (c *Controller) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.isRunning {
		return
	}

	c.isRunning = false
	close(c.quit)
	if c.ticker != nil {
		c.ticker.Stop()
	}

	go c.tick()
}

func (c *Controller) run() {
	initSchedules := <-c.mch.Data
	c.update(initSchedules)

	for {
		select {
		case <-c.quit:
			close(c.mch.Quit)
			return
		case <-c.ticker.C():
			c.tick()
		case scheds := <-c.mch.Data:
			c.update(scheds)
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

	if c.done != nil {
		c.done <- struct{}{}
	}
}

func (c *Controller) update(schedules []*pb.MetricConfigResponse_Schedule) {
	c.matcher.ConsumeSchedules(schedules)
	minPeriod := c.matcher.GetMinPeriod()
	if c.lastPeriod != minPeriod {
		if c.ticker != nil {
			c.ticker.Stop()
		}

		c.lastPeriod = minPeriod

		// TOOD: create tidier, encapsulated dynamic ticker
		c.lock.Lock()
		c.ticker = c.clock.Ticker(c.lastPeriod)
		c.lock.Unlock()

		if c.done != nil {
			c.done <- struct{}{}
		}
	}
}

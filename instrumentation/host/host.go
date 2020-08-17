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

package host // import "go.opentelemetry.io/contrib/instrumentation/host"

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"

	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
)

// Host reports the work-in-progress conventional host metrics specified by OpenTelemetry
type host struct {
	config Config
	meter  metric.Meter
}

// Config contains optional settings for reporting host metrics.
type Config struct {
	// MeterProvider sets the metric.Provider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.Provider
}

// Option supports configuring optional settings for host metrics.
type Option interface {
	// ApplyHost updates *Config.
	ApplyHost(*Config)
}

// WithMeterProvider sets the Metric implementation to use for
// reporting.  If this option is not used, the global metric.Provider
// will be used.  `provider` must be non-nil.
func WithMeterProvider(provider metric.Provider) Option {
	return metricProviderOption{provider}
}

type metricProviderOption struct{ metric.Provider }

// ApplyHost implements Option.
func (o metricProviderOption) ApplyHost(c *Config) {
	c.MeterProvider = o.Provider
}

var (
	LabelCPUTimeUser   = []kv.KeyValue{kv.String("state", "user")}
	LabelCPUTimeSystem = []kv.KeyValue{kv.String("state", "system")}
	LabelCPUTimeOther  = []kv.KeyValue{kv.String("state", "other")}
	LabelCPUTimeIdle   = []kv.KeyValue{kv.String("state", "idle")}
)

// Configure computes a Config from the supplied Options.
func Configure(opts ...Option) Config {
	c := Config{
		MeterProvider: global.MeterProvider(),
	}
	for _, opt := range opts {
		opt.ApplyHost(&c)
	}
	return c
}

// Start initializes reporting of host metrics using the supplied Config.
func Start(c Config) error {
	if c.MeterProvider == nil {
		c.MeterProvider = global.MeterProvider()
	}
	h := &host{
		meter: c.MeterProvider.Meter(
			"host",
			metric.WithInstrumentationVersion(contrib.SemVersion()),
		),
		config: c,
	}
	return h.register()
}

func (h *host) register() error {
	var (
		err error

		processCPUTime metric.Float64SumObserver
		hostCPUTime    metric.Float64SumObserver

		// lock prevents a race between batch observer and instrument registration.
		lock sync.Mutex
	)

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return fmt.Errorf("could not find this process: %w", err)
	}

	lock.Lock()
	defer lock.Unlock()

	batchObserver := h.meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		lock.Lock()
		defer lock.Unlock()

		// This follows the OpenTelemetry Collector's "hostmetrics"
		// receiver/hostmetricsreceiver/internal/scraper/processscraper
		// measures User and System IOwait time.
		// TODO: the Collector has per-OS compilation modules to support
		// specific metrics that are not universal.
		processTimes, err := proc.TimesWithContext(ctx)
		if err != nil {
			global.Handler().Handle(err)
			return
		}

		hostTimeSlice, err := cpu.TimesWithContext(ctx, false)
		if err != nil {
			global.Handler().Handle(err)
			return
		}
		if len(hostTimeSlice) != 1 {
			global.Handler().Handle(fmt.Errorf("host CPU usage: incorrect summary count"))
			return
		}

		result.Observe(LabelCPUTimeUser, processCPUTime.Observation(processTimes.User))
		result.Observe(LabelCPUTimeSystem, processCPUTime.Observation(processTimes.System))

		hostTime := hostTimeSlice[0]
		result.Observe(LabelCPUTimeUser, hostCPUTime.Observation(hostTime.User))
		result.Observe(LabelCPUTimeSystem, hostCPUTime.Observation(hostTime.System))

		other := hostTime.Nice +
			hostTime.Iowait +
			hostTime.Irq +
			hostTime.Softirq +
			hostTime.Steal +
			hostTime.Guest +
			hostTime.GuestNice

		result.Observe(LabelCPUTimeOther, hostCPUTime.Observation(other))
		result.Observe(LabelCPUTimeIdle, hostCPUTime.Observation(hostTime.Idle))
	})

	// Note: Units are in seconds, but "unit" package does not
	// include this string.
	if processCPUTime, err = batchObserver.NewFloat64SumObserver(
		"process.cpu.time",
		metric.WithUnit("s"),
		metric.WithDescription(
			"Accumulated CPU time spent by this process labeled with attribution (User, System, ...)",
		),
	); err != nil {
		return err
	}

	if hostCPUTime, err = batchObserver.NewFloat64SumObserver(
		"system.cpu.time",
		metric.WithUnit("s"),
		metric.WithDescription(
			"Accumulated CPU time spent by this host labeled with attribution (User, System, ...)",
		),
	); err != nil {
		return err
	}

	return nil
}

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

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/unit"
)

// Host reports the work-in-progress conventional host metrics specified by OpenTelemetry
type host struct {
	config config
	meter  metric.Meter
}

// config contains optional settings for reporting host metrics.
type config struct {
	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider
}

// Option supports configuring optional settings for host metrics.
type Option interface {
	apply(*config)
}

// WithMeterProvider sets the Metric implementation to use for
// reporting.  If this option is not used, the global metric.MeterProvider
// will be used.  `provider` must be non-nil.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return metricProviderOption{provider}
}

type metricProviderOption struct{ metric.MeterProvider }

func (o metricProviderOption) apply(c *config) {
	if o.MeterProvider != nil {
		c.MeterProvider = o.MeterProvider
	}
}

// Attribute sets.
var (
	// Attribute sets for CPU time measurements.

	AttributeCPUTimeUser   = []attribute.KeyValue{attribute.String("state", "user")}
	AttributeCPUTimeSystem = []attribute.KeyValue{attribute.String("state", "system")}
	AttributeCPUTimeOther  = []attribute.KeyValue{attribute.String("state", "other")}
	AttributeCPUTimeIdle   = []attribute.KeyValue{attribute.String("state", "idle")}

	// Attribute sets used for Memory measurements.

	AttributeMemoryAvailable = []attribute.KeyValue{attribute.String("state", "available")}
	AttributeMemoryUsed      = []attribute.KeyValue{attribute.String("state", "used")}

	// Attribute sets used for Network measurements.

	AttributeNetworkTransmit = []attribute.KeyValue{attribute.String("direction", "transmit")}
	AttributeNetworkReceive  = []attribute.KeyValue{attribute.String("direction", "receive")}
)

// newConfig computes a config from a list of Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider: global.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	return c
}

// Start initializes reporting of host metrics using the supplied config.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	if c.MeterProvider == nil {
		c.MeterProvider = global.GetMeterProvider()
	}
	h := &host{
		meter: c.MeterProvider.Meter(
			"go.opentelemetry.io/contrib/instrumentation/host",
			metric.WithInstrumentationVersion(SemVersion()),
		),
		config: c,
	}
	return h.register()
}

func (h *host) register() error {
	var (
		err error

		processCPUTime metric.Float64CounterObserver
		hostCPUTime    metric.Float64CounterObserver

		hostMemoryUsage       metric.Int64GaugeObserver
		hostMemoryUtilization metric.Float64GaugeObserver

		networkIOUsage metric.Int64CounterObserver

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
			otel.Handle(err)
			return
		}

		hostTimeSlice, err := cpu.TimesWithContext(ctx, false)
		if err != nil {
			otel.Handle(err)
			return
		}
		if len(hostTimeSlice) != 1 {
			otel.Handle(fmt.Errorf("host CPU usage: incorrect summary count"))
			return
		}

		vmStats, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			otel.Handle(err)
			return
		}

		ioStats, err := net.IOCountersWithContext(ctx, false)
		if err != nil {
			otel.Handle(err)
			return
		}
		if len(ioStats) != 1 {
			otel.Handle(fmt.Errorf("host network usage: incorrect summary count"))
			return
		}

		// Process CPU time
		result.Observe(AttributeCPUTimeUser, processCPUTime.Observation(processTimes.User))
		result.Observe(AttributeCPUTimeSystem, processCPUTime.Observation(processTimes.System))

		// Host CPU time
		hostTime := hostTimeSlice[0]
		result.Observe(AttributeCPUTimeUser, hostCPUTime.Observation(hostTime.User))
		result.Observe(AttributeCPUTimeSystem, hostCPUTime.Observation(hostTime.System))

		// TODO(#244): "other" is a placeholder for actually dealing
		// with these states.  Do users actually want this
		// (unconditionally)?  How should we handle "iowait"
		// if not all systems expose it?  Should we break
		// these down by CPU?  If so, are users going to want
		// to aggregate in-process?  See:
		// https://github.com/open-telemetry/opentelemetry-go-contrib/issues/244
		other := hostTime.Nice +
			hostTime.Iowait +
			hostTime.Irq +
			hostTime.Softirq +
			hostTime.Steal +
			hostTime.Guest +
			hostTime.GuestNice

		result.Observe(AttributeCPUTimeOther, hostCPUTime.Observation(other))
		result.Observe(AttributeCPUTimeIdle, hostCPUTime.Observation(hostTime.Idle))

		// Host memory usage
		result.Observe(AttributeMemoryUsed, hostMemoryUsage.Observation(int64(vmStats.Used)))
		result.Observe(AttributeMemoryAvailable, hostMemoryUsage.Observation(int64(vmStats.Available)))

		// Host memory utilization
		result.Observe(AttributeMemoryUsed,
			hostMemoryUtilization.Observation(float64(vmStats.Used)/float64(vmStats.Total)),
		)
		result.Observe(AttributeMemoryAvailable,
			hostMemoryUtilization.Observation(float64(vmStats.Available)/float64(vmStats.Total)),
		)

		// Host network usage
		//
		// TODO: These can be broken down by network
		// interface, with similar questions to those posed
		// about per-CPU measurements above.
		result.Observe(AttributeNetworkTransmit, networkIOUsage.Observation(int64(ioStats[0].BytesSent)))
		result.Observe(AttributeNetworkReceive, networkIOUsage.Observation(int64(ioStats[0].BytesRecv)))
	})

	// TODO: .time units are in seconds, but "unit" package does
	// not include this string.
	// https://github.com/open-telemetry/opentelemetry-specification/issues/705
	if processCPUTime, err = batchObserver.NewFloat64CounterObserver(
		"process.cpu.time",
		metric.WithUnit("s"),
		metric.WithDescription(
			"Accumulated CPU time spent by this process attributeed by state (User, System, ...)",
		),
	); err != nil {
		return err
	}

	if hostCPUTime, err = batchObserver.NewFloat64CounterObserver(
		"system.cpu.time",
		metric.WithUnit("s"),
		metric.WithDescription(
			"Accumulated CPU time spent by this host attributeed by state (User, System, Other, Idle)",
		),
	); err != nil {
		return err
	}

	if hostMemoryUsage, err = batchObserver.NewInt64GaugeObserver(
		"system.memory.usage",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription(
			"Memory usage of this process attributed by memory state (Used, Available)",
		),
	); err != nil {
		return err
	}

	if hostMemoryUtilization, err = batchObserver.NewFloat64GaugeObserver(
		"system.memory.utilization",
		metric.WithUnit(unit.Dimensionless),
		metric.WithDescription(
			"Memory utilization of this process attributeed by memory state (Used, Available)",
		),
	); err != nil {
		return err
	}

	if networkIOUsage, err = batchObserver.NewInt64CounterObserver(
		"system.network.io",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription(
			"Bytes transferred attributeed by direction (Transmit, Receive)",
		),
	); err != nil {
		return err
	}

	return nil
}

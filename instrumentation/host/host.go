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
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/unit"
)

// Host reports the work-in-progress conventional host metrics specified by OpenTelemetry.
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
		MeterProvider: global.MeterProvider(),
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
		c.MeterProvider = global.MeterProvider()
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

		processCPUTime asyncfloat64.Counter
		hostCPUTime    asyncfloat64.Counter

		hostMemoryUsage       asyncint64.Gauge
		hostMemoryUtilization asyncfloat64.Gauge

		networkIOUsage asyncint64.Counter

		// lock prevents a race between batch observer and instrument registration.
		lock sync.Mutex
	)

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return fmt.Errorf("could not find this process: %w", err)
	}

	lock.Lock()
	defer lock.Unlock()

	// TODO: .time units are in seconds, but "unit" package does
	// not include this string.
	// https://github.com/open-telemetry/opentelemetry-specification/issues/705
	if processCPUTime, err = h.meter.AsyncFloat64().Counter(
		"process.cpu.time",
		instrument.WithUnit("s"),
		instrument.WithDescription(
			"Accumulated CPU time spent by this process attributeed by state (User, System, ...)",
		),
	); err != nil {
		return err
	}

	if hostCPUTime, err = h.meter.AsyncFloat64().Counter(
		"system.cpu.time",
		instrument.WithUnit("s"),
		instrument.WithDescription(
			"Accumulated CPU time spent by this host attributeed by state (User, System, Other, Idle)",
		),
	); err != nil {
		return err
	}

	if hostMemoryUsage, err = h.meter.AsyncInt64().Gauge(
		"system.memory.usage",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription(
			"Memory usage of this process attributed by memory state (Used, Available)",
		),
	); err != nil {
		return err
	}

	if hostMemoryUtilization, err = h.meter.AsyncFloat64().Gauge(
		"system.memory.utilization",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription(
			"Memory utilization of this process attributeed by memory state (Used, Available)",
		),
	); err != nil {
		return err
	}

	if networkIOUsage, err = h.meter.AsyncInt64().Counter(
		"system.network.io",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription(
			"Bytes transferred attributeed by direction (Transmit, Receive)",
		),
	); err != nil {
		return err
	}

	err = h.meter.RegisterCallback(
		[]instrument.Asynchronous{
			processCPUTime,
			hostCPUTime,
			hostMemoryUsage,
			hostMemoryUtilization,
			networkIOUsage,
		},
		func(ctx context.Context) {
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
			processCPUTime.Observe(ctx, processTimes.User, AttributeCPUTimeUser...)
			processCPUTime.Observe(ctx, processTimes.System, AttributeCPUTimeSystem...)

			// Host CPU time
			hostTime := hostTimeSlice[0]
			hostCPUTime.Observe(ctx, hostTime.User, AttributeCPUTimeUser...)
			hostCPUTime.Observe(ctx, hostTime.System, AttributeCPUTimeSystem...)

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

			hostCPUTime.Observe(ctx, other, AttributeCPUTimeOther...)
			hostCPUTime.Observe(ctx, hostTime.Idle, AttributeCPUTimeIdle...)

			// Host memory usage
			hostMemoryUsage.Observe(ctx, int64(vmStats.Used), AttributeMemoryUsed...)
			hostMemoryUsage.Observe(ctx, int64(vmStats.Available), AttributeMemoryAvailable...)

			// Host memory utilization
			hostMemoryUtilization.Observe(ctx, float64(vmStats.Used)/float64(vmStats.Total), AttributeMemoryUsed...)
			hostMemoryUtilization.Observe(ctx, float64(vmStats.Available)/float64(vmStats.Total), AttributeMemoryAvailable...)

			// Host network usage
			//
			// TODO: These can be broken down by network
			// interface, with similar questions to those posed
			// about per-CPU measurements above.
			networkIOUsage.Observe(ctx, int64(ioStats[0].BytesSent), AttributeNetworkTransmit...)
			networkIOUsage.Observe(ctx, int64(ioStats[0].BytesRecv), AttributeNetworkReceive...)
		})

	if err != nil {
		return err
	}

	return nil
}

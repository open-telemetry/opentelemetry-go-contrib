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
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/host"

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

	AttributeCPUTimeUser   = attribute.NewSet(attribute.String("state", "user"))
	AttributeCPUTimeSystem = attribute.NewSet(attribute.String("state", "system"))
	AttributeCPUTimeOther  = attribute.NewSet(attribute.String("state", "other"))
	AttributeCPUTimeIdle   = attribute.NewSet(attribute.String("state", "idle"))

	// Attribute sets used for Memory measurements.

	AttributeMemoryAvailable = attribute.NewSet(attribute.String("state", "available"))
	AttributeMemoryUsed      = attribute.NewSet(attribute.String("state", "used"))

	// Attribute sets used for Network measurements.

	AttributeNetworkTransmit = attribute.NewSet(attribute.String("direction", "transmit"))
	AttributeNetworkReceive  = attribute.NewSet(attribute.String("direction", "receive"))
)

// newConfig computes a config from a list of Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider: otel.GetMeterProvider(),
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
		c.MeterProvider = otel.GetMeterProvider()
	}
	h := &host{
		meter: c.MeterProvider.Meter(
			ScopeName,
			metric.WithInstrumentationVersion(Version()),
		),
		config: c,
	}
	return h.register()
}

func (h *host) register() error {
	var (
		err error

		processCPUTime metric.Float64ObservableCounter
		hostCPUTime    metric.Float64ObservableCounter

		hostMemoryUsage       metric.Int64ObservableGauge
		hostMemoryUtilization metric.Float64ObservableGauge

		networkIOUsage metric.Int64ObservableCounter

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
	if processCPUTime, err = h.meter.Float64ObservableCounter(
		"process.cpu.time",
		metric.WithUnit("s"),
		metric.WithDescription(
			"Accumulated CPU time spent by this process attributed by state (User, System, ...)",
		),
	); err != nil {
		return err
	}

	if hostCPUTime, err = h.meter.Float64ObservableCounter(
		"system.cpu.time",
		metric.WithUnit("s"),
		metric.WithDescription(
			"Accumulated CPU time spent by this host attributed by state (User, System, Other, Idle)",
		),
	); err != nil {
		return err
	}

	if hostMemoryUsage, err = h.meter.Int64ObservableGauge(
		"system.memory.usage",
		metric.WithUnit("By"),
		metric.WithDescription(
			"Memory usage of this process attributed by memory state (Used, Available)",
		),
	); err != nil {
		return err
	}

	if hostMemoryUtilization, err = h.meter.Float64ObservableGauge(
		"system.memory.utilization",
		metric.WithUnit("1"),
		metric.WithDescription(
			"Memory utilization of this process attributed by memory state (Used, Available)",
		),
	); err != nil {
		return err
	}

	if networkIOUsage, err = h.meter.Int64ObservableCounter(
		"system.network.io",
		metric.WithUnit("By"),
		metric.WithDescription(
			"Bytes transferred attributed by direction (Transmit, Receive)",
		),
	); err != nil {
		return err
	}

	_, err = h.meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			lock.Lock()
			defer lock.Unlock()

			// This follows the OpenTelemetry Collector's "hostmetrics"
			// receiver/hostmetricsreceiver/internal/scraper/processscraper
			// measures User and System IOwait time.
			// TODO: the Collector has per-OS compilation modules to support
			// specific metrics that are not universal.
			processTimes, err := proc.TimesWithContext(ctx)
			if err != nil {
				return err
			}

			hostTimeSlice, err := cpu.TimesWithContext(ctx, false)
			if err != nil {
				return err
			}
			if len(hostTimeSlice) != 1 {
				return fmt.Errorf("host CPU usage: incorrect summary count")
			}

			vmStats, err := mem.VirtualMemoryWithContext(ctx)
			if err != nil {
				return err
			}

			ioStats, err := net.IOCountersWithContext(ctx, false)
			if err != nil {
				return err
			}
			if len(ioStats) != 1 {
				return fmt.Errorf("host network usage: incorrect summary count")
			}

			hostTime := hostTimeSlice[0]
			opt := metric.WithAttributeSet(AttributeCPUTimeUser)
			o.ObserveFloat64(processCPUTime, processTimes.User, opt)
			o.ObserveFloat64(hostCPUTime, hostTime.User, opt)

			opt = metric.WithAttributeSet(AttributeCPUTimeSystem)
			o.ObserveFloat64(processCPUTime, processTimes.System, opt)
			o.ObserveFloat64(hostCPUTime, hostTime.System, opt)

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

			opt = metric.WithAttributeSet(AttributeCPUTimeOther)
			o.ObserveFloat64(hostCPUTime, other, opt)
			opt = metric.WithAttributeSet(AttributeCPUTimeIdle)
			o.ObserveFloat64(hostCPUTime, hostTime.Idle, opt)

			// Host memory usage
			opt = metric.WithAttributeSet(AttributeMemoryUsed)
			o.ObserveInt64(hostMemoryUsage, int64(vmStats.Used), opt)
			opt = metric.WithAttributeSet(AttributeMemoryAvailable)
			o.ObserveInt64(hostMemoryUsage, int64(vmStats.Available), opt)

			// Host memory utilization
			opt = metric.WithAttributeSet(AttributeMemoryUsed)
			o.ObserveFloat64(hostMemoryUtilization, float64(vmStats.Used)/float64(vmStats.Total), opt)
			opt = metric.WithAttributeSet(AttributeMemoryAvailable)
			o.ObserveFloat64(hostMemoryUtilization, float64(vmStats.Available)/float64(vmStats.Total), opt)

			// Host network usage
			//
			// TODO: These can be broken down by network
			// interface, with similar questions to those posed
			// about per-CPU measurements above.
			opt = metric.WithAttributeSet(AttributeNetworkTransmit)
			o.ObserveInt64(networkIOUsage, int64(ioStats[0].BytesSent), opt)
			opt = metric.WithAttributeSet(AttributeNetworkReceive)
			o.ObserveInt64(networkIOUsage, int64(ioStats[0].BytesRecv), opt)

			return nil
		},
		processCPUTime,
		hostCPUTime,
		hostMemoryUsage,
		hostMemoryUtilization,
		networkIOUsage,
	)

	if err != nil {
		return err
	}

	return nil
}

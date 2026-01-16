// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package host // import "go.opentelemetry.io/contrib/instrumentation/host"

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.37.0/processconv"
	"go.opentelemetry.io/otel/semconv/v1.37.0/systemconv"
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

	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeCPUTimeUser = attribute.NewSet(attribute.String("state", "user"))
	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeCPUTimeSystem = attribute.NewSet(attribute.String("state", "system"))
	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeCPUTimeOther = attribute.NewSet(attribute.String("state", "other"))
	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeCPUTimeIdle = attribute.NewSet(attribute.String("state", "idle"))

	// Attribute sets used for Memory measurements.

	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeMemoryAvailable = attribute.NewSet(attribute.String("state", "available"))
	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeMemoryUsed = attribute.NewSet(attribute.String("state", "used"))

	// Attribute sets used for Network measurements.

	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeNetworkTransmit = attribute.NewSet(attribute.String("direction", "transmit"))
	// Deprecated: Use go.opentelemetry.io/otel/semconv instead.
	AttributeNetworkReceive = attribute.NewSet(attribute.String("direction", "receive"))
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
			metric.WithInstrumentationVersion(Version),
		),
		config: c,
	}
	return h.register()
}

func (h *host) register() error {
	var (
		err error

		procCPUTime         processconv.CPUTime
		procCPUTimeModeUser = metric.WithAttributes(
			procCPUTime.AttrCPUMode(processconv.CPUModeUser),
		)
		procCPUTimeModeSystem = metric.WithAttributes(
			procCPUTime.AttrCPUMode(processconv.CPUModeSystem),
		)

		cpuTime         systemconv.CPUTime
		cpuTimeModeUser = metric.WithAttributes(
			cpuTime.AttrCPUMode(systemconv.CPUModeUser),
		)
		cpuTimeModeSystem = metric.WithAttributes(
			cpuTime.AttrCPUMode(systemconv.CPUModeSystem),
		)
		cpuTimeModeIdle = metric.WithAttributes(
			cpuTime.AttrCPUMode(systemconv.CPUModeIdle),
		)
		cpuTimeModeOther = metric.WithAttributes(
			cpuTime.AttrCPUMode(systemconv.CPUModeAttr("other")),
		)

		memUse          systemconv.MemoryUsage
		memUseStateFree = metric.WithAttributes(
			memUse.AttrMemoryState(systemconv.MemoryStateFree),
		)
		memUseStateUsed = metric.WithAttributes(
			memUse.AttrMemoryState(systemconv.MemoryStateUsed),
		)

		memUtil          systemconv.MemoryUtilization
		memUtilStateFree = metric.WithAttributes(
			memUtil.AttrMemoryState(systemconv.MemoryStateFree),
		)
		memUtilStateUsed = metric.WithAttributes(
			memUtil.AttrMemoryState(systemconv.MemoryStateUsed),
		)

		netIO              systemconv.NetworkIO
		netIOStateTransmit = metric.WithAttributes(
			netIO.AttrNetworkIODirection(systemconv.NetworkIODirectionTransmit),
		)
		netIOStateReceive = metric.WithAttributes(
			netIO.AttrNetworkIODirection(systemconv.NetworkIODirectionReceive),
		)

		// lock prevents a race between batch observer and instrument registration.
		lock sync.Mutex
	)

	pid := os.Getpid()
	if pid > math.MaxInt32 || pid < math.MinInt32 {
		return fmt.Errorf("invalid process ID: %d", pid)
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return fmt.Errorf("could not find this process: %w", err)
	}

	lock.Lock()
	defer lock.Unlock()

	if procCPUTime, err = processconv.NewCPUTime(h.meter); err != nil {
		return err
	}
	if cpuTime, err = systemconv.NewCPUTime(h.meter); err != nil {
		return err
	}
	if memUse, err = systemconv.NewMemoryUsage(h.meter); err != nil {
		return err
	}
	if memUtil, err = systemconv.NewMemoryUtilization(h.meter); err != nil {
		return err
	}
	if netIO, err = systemconv.NewNetworkIO(h.meter); err != nil {
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
				return errors.New("host CPU usage: incorrect summary count")
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
				return errors.New("host network usage: incorrect summary count")
			}

			hostTime := hostTimeSlice[0]
			o.ObserveFloat64(procCPUTime.Inst(), processTimes.User, procCPUTimeModeUser)
			o.ObserveFloat64(procCPUTime.Inst(), processTimes.System, procCPUTimeModeSystem)

			o.ObserveFloat64(cpuTime.Inst(), hostTime.User, cpuTimeModeUser)
			o.ObserveFloat64(cpuTime.Inst(), hostTime.System, cpuTimeModeSystem)

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

			o.ObserveFloat64(cpuTime.Inst(), other, cpuTimeModeOther)
			o.ObserveFloat64(cpuTime.Inst(), hostTime.Idle, cpuTimeModeIdle)

			// Host memory usage
			o.ObserveInt64(memUse.Inst(), clampInt64(vmStats.Used), memUseStateUsed)
			o.ObserveInt64(memUse.Inst(), clampInt64(vmStats.Available), memUseStateFree)

			// Host memory utilization
			o.ObserveFloat64(
				memUtil.Inst(),
				float64(vmStats.Used)/float64(vmStats.Total), memUtilStateUsed,
			)
			o.ObserveFloat64(
				memUtil.Inst(),
				float64(vmStats.Available)/float64(vmStats.Total),
				memUtilStateFree,
			)

			// Host network usage
			//
			// TODO: These can be broken down by network
			// interface, with similar questions to those posed
			// about per-CPU measurements above.
			o.ObserveInt64(
				netIO.Inst(),
				clampInt64(ioStats[0].BytesSent),
				netIOStateTransmit,
			)
			o.ObserveInt64(
				netIO.Inst(),
				clampInt64(ioStats[0].BytesRecv),
				netIOStateReceive,
			)

			return nil
		},
		procCPUTime.Inst(),
		cpuTime.Inst(),
		memUse.Inst(),
		memUtil.Inst(),
		netIO.Inst(),
	)
	if err != nil {
		return err
	}

	return nil
}

func clampInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

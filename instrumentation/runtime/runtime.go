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

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"context"
	goruntime "runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
)

// Runtime reports the work-in-progress conventional runtime metrics specified by OpenTelemetry.
type runtime struct {
	config config
	meter  metric.Meter
}

// config contains optional settings for reporting runtime metrics.
type config struct {
	// MinimumReadMemStatsInterval sets the mininum interval
	// between calls to runtime.ReadMemStats().  Negative values
	// are ignored.
	MinimumReadMemStatsInterval time.Duration

	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider
}

// Option supports configuring optional settings for runtime metrics.
type Option interface {
	apply(*config)
}

// DefaultMinimumReadMemStatsInterval is the default minimum interval
// between calls to runtime.ReadMemStats().  Use the
// WithMinimumReadMemStatsInterval() option to modify this setting in
// Start().
const DefaultMinimumReadMemStatsInterval time.Duration = 15 * time.Second

// WithMinimumReadMemStatsInterval sets a minimum interval between calls to
// runtime.ReadMemStats(), which is a relatively expensive call to make
// frequently.  This setting is ignored when `d` is negative.
func WithMinimumReadMemStatsInterval(d time.Duration) Option {
	return minimumReadMemStatsIntervalOption(d)
}

type minimumReadMemStatsIntervalOption time.Duration

func (o minimumReadMemStatsIntervalOption) apply(c *config) {
	if o >= 0 {
		c.MinimumReadMemStatsInterval = time.Duration(o)
	}
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

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider:               global.MeterProvider(),
		MinimumReadMemStatsInterval: DefaultMinimumReadMemStatsInterval,
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	return c
}

// Start initializes reporting of runtime metrics using the supplied config.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	if c.MinimumReadMemStatsInterval < 0 {
		c.MinimumReadMemStatsInterval = DefaultMinimumReadMemStatsInterval
	}
	if c.MeterProvider == nil {
		c.MeterProvider = global.MeterProvider()
	}
	r := &runtime{
		meter: c.MeterProvider.Meter(
			"go.opentelemetry.io/contrib/instrumentation/runtime",
			metric.WithInstrumentationVersion(SemVersion()),
		),
		config: c,
	}
	return r.register()
}

func (r *runtime) register() error {
	startTime := time.Now()
	uptime, err := r.meter.Int64ObservableUpDownCounter(
		"runtime.uptime",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("Milliseconds since application was initialized"),
	)
	if err != nil {
		return err
	}

	goroutines, err := r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.goroutines",
		instrument.WithDescription("Number of goroutines that currently exist"),
	)
	if err != nil {
		return err
	}

	cgoCalls, err := r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.cgo.calls",
		instrument.WithDescription("Number of cgo calls made by the current process"),
	)
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			o.ObserveInt64(uptime, time.Since(startTime).Milliseconds())
			o.ObserveInt64(goroutines, int64(goruntime.NumGoroutine()))
			o.ObserveInt64(cgoCalls, goruntime.NumCgoCall())
			return nil
		},
		uptime,
		goroutines,
		cgoCalls,
	)
	if err != nil {
		return err
	}

	return r.registerMemStats()
}

func (r *runtime) registerMemStats() error {
	var (
		err error

		heapAlloc    instrument.Int64ObservableUpDownCounter
		heapIdle     instrument.Int64ObservableUpDownCounter
		heapInuse    instrument.Int64ObservableUpDownCounter
		heapObjects  instrument.Int64ObservableUpDownCounter
		heapReleased instrument.Int64ObservableUpDownCounter
		heapSys      instrument.Int64ObservableUpDownCounter
		liveObjects  instrument.Int64ObservableUpDownCounter

		// TODO: is ptrLookups useful? I've not seen a value
		// other than zero.
		ptrLookups instrument.Int64ObservableCounter

		gcCount      instrument.Int64ObservableCounter
		pauseTotalNs instrument.Int64ObservableCounter
		gcPauseNs    instrument.Int64Histogram

		lastNumGC    uint32
		lastMemStats time.Time
		memStats     goruntime.MemStats

		// lock prevents a race between batch observer and instrument registration.
		lock sync.Mutex
	)

	lock.Lock()
	defer lock.Unlock()

	if heapAlloc, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.heap_alloc",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes of allocated heap objects"),
	); err != nil {
		return err
	}

	if heapIdle, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.heap_idle",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes in idle (unused) spans"),
	); err != nil {
		return err
	}

	if heapInuse, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.heap_inuse",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes in in-use spans"),
	); err != nil {
		return err
	}

	if heapObjects, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.heap_objects",
		instrument.WithDescription("Number of allocated heap objects"),
	); err != nil {
		return err
	}

	// FYI see https://github.com/golang/go/issues/32284 to help
	// understand the meaning of this value.
	if heapReleased, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.heap_released",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes of idle spans whose physical memory has been returned to the OS"),
	); err != nil {
		return err
	}

	if heapSys, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.heap_sys",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes of heap memory obtained from the OS"),
	); err != nil {
		return err
	}

	if ptrLookups, err = r.meter.Int64ObservableCounter(
		"process.runtime.go.mem.lookups",
		instrument.WithDescription("Number of pointer lookups performed by the runtime"),
	); err != nil {
		return err
	}

	if liveObjects, err = r.meter.Int64ObservableUpDownCounter(
		"process.runtime.go.mem.live_objects",
		instrument.WithDescription("Number of live objects is the number of cumulative Mallocs - Frees"),
	); err != nil {
		return err
	}

	if gcCount, err = r.meter.Int64ObservableCounter(
		"process.runtime.go.gc.count",
		instrument.WithDescription("Number of completed garbage collection cycles"),
	); err != nil {
		return err
	}

	// Note that the following could be derived as a sum of
	// individual pauses, but we may lose individual pauses if the
	// observation interval is too slow.
	if pauseTotalNs, err = r.meter.Int64ObservableCounter(
		"process.runtime.go.gc.pause_total_ns",
		// TODO: nanoseconds units
		instrument.WithDescription("Cumulative nanoseconds in GC stop-the-world pauses since the program started"),
	); err != nil {
		return err
	}

	if gcPauseNs, err = r.meter.Int64Histogram(
		"process.runtime.go.gc.pause_ns",
		// TODO: nanoseconds units
		instrument.WithDescription("Amount of nanoseconds in GC stop-the-world pauses"),
	); err != nil {
		return err
	}

	_, err = r.meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			lock.Lock()
			defer lock.Unlock()

			now := time.Now()
			if now.Sub(lastMemStats) >= r.config.MinimumReadMemStatsInterval {
				goruntime.ReadMemStats(&memStats)
				lastMemStats = now
			}

			o.ObserveInt64(heapAlloc, int64(memStats.HeapAlloc))
			o.ObserveInt64(heapIdle, int64(memStats.HeapIdle))
			o.ObserveInt64(heapInuse, int64(memStats.HeapInuse))
			o.ObserveInt64(heapObjects, int64(memStats.HeapObjects))
			o.ObserveInt64(heapReleased, int64(memStats.HeapReleased))
			o.ObserveInt64(heapSys, int64(memStats.HeapSys))
			o.ObserveInt64(liveObjects, int64(memStats.Mallocs-memStats.Frees))
			o.ObserveInt64(ptrLookups, int64(memStats.Lookups))
			o.ObserveInt64(gcCount, int64(memStats.NumGC))
			o.ObserveInt64(pauseTotalNs, int64(memStats.PauseTotalNs))

			computeGCPauses(ctx, gcPauseNs, memStats.PauseNs[:], lastNumGC, memStats.NumGC)

			lastNumGC = memStats.NumGC

			return nil
		},
		heapAlloc,
		heapIdle,
		heapInuse,
		heapObjects,
		heapReleased,
		heapSys,
		liveObjects,

		ptrLookups,

		gcCount,
		pauseTotalNs,
	)
	if err != nil {
		return err
	}
	return nil
}

func computeGCPauses(
	ctx context.Context,
	recorder instrument.Int64Histogram,
	circular []uint64,
	lastNumGC, currentNumGC uint32,
) {
	delta := int(int64(currentNumGC) - int64(lastNumGC))

	if delta == 0 {
		return
	}

	if delta >= len(circular) {
		// There were > 256 collections, some may have been lost.
		recordGCPauses(ctx, recorder, circular)
		return
	}

	length := uint32(len(circular))

	i := lastNumGC % length
	j := currentNumGC % length

	if j < i { // wrap around the circular buffer
		recordGCPauses(ctx, recorder, circular[i:])
		recordGCPauses(ctx, recorder, circular[:j])
		return
	}

	recordGCPauses(ctx, recorder, circular[i:j])
}

func recordGCPauses(
	ctx context.Context,
	recorder instrument.Int64Histogram,
	pauses []uint64,
) {
	for _, pause := range pauses {
		recorder.Record(ctx, int64(pause))
	}
}

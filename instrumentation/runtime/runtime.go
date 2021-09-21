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
	"go.opentelemetry.io/otel/metric/unit"
)

// Runtime reports the work-in-progress conventional runtime metrics specified by OpenTelemetry
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
		MeterProvider:               global.GetMeterProvider(),
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
		c.MeterProvider = global.GetMeterProvider()
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
	if _, err := r.meter.NewInt64CounterObserver(
		"runtime.uptime",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(time.Since(startTime).Milliseconds())
		},
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("Milliseconds since application was initialized"),
	); err != nil {
		return err
	}

	if _, err := r.meter.NewInt64UpDownCounterObserver(
		"runtime.go.goroutines",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(int64(goruntime.NumGoroutine()))
		},
		metric.WithDescription("Number of goroutines that currently exist"),
	); err != nil {
		return err
	}

	if _, err := r.meter.NewInt64CounterObserver(
		"runtime.go.cgo.calls",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(goruntime.NumCgoCall())
		},
		metric.WithDescription("Number of cgo calls made by the current process"),
	); err != nil {
		return err
	}

	return r.registerMemStats()
}

func (r *runtime) registerMemStats() error {
	var (
		err error

		heapAlloc    metric.Int64UpDownCounterObserver
		heapIdle     metric.Int64UpDownCounterObserver
		heapInuse    metric.Int64UpDownCounterObserver
		heapObjects  metric.Int64UpDownCounterObserver
		heapReleased metric.Int64UpDownCounterObserver
		heapSys      metric.Int64UpDownCounterObserver
		liveObjects  metric.Int64UpDownCounterObserver

		// TODO: is ptrLookups useful? I've not seen a value
		// other than zero.
		ptrLookups metric.Int64CounterObserver

		gcCount      metric.Int64CounterObserver
		pauseTotalNs metric.Int64CounterObserver
		gcPauseNs    metric.Int64Histogram

		lastNumGC    uint32
		lastMemStats time.Time
		memStats     goruntime.MemStats

		// lock prevents a race between batch observer and instrument registration.
		lock sync.Mutex
	)

	lock.Lock()
	defer lock.Unlock()

	batchObserver := r.meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		lock.Lock()
		defer lock.Unlock()

		now := time.Now()
		if now.Sub(lastMemStats) >= r.config.MinimumReadMemStatsInterval {
			goruntime.ReadMemStats(&memStats)
			lastMemStats = now
		}

		result.Observe(
			nil,
			heapAlloc.Observation(int64(memStats.HeapAlloc)),
			heapIdle.Observation(int64(memStats.HeapIdle)),
			heapInuse.Observation(int64(memStats.HeapInuse)),
			heapObjects.Observation(int64(memStats.HeapObjects)),
			heapReleased.Observation(int64(memStats.HeapReleased)),
			heapSys.Observation(int64(memStats.HeapSys)),
			liveObjects.Observation(int64(memStats.Mallocs-memStats.Frees)),
			ptrLookups.Observation(int64(memStats.Lookups)),
			gcCount.Observation(int64(memStats.NumGC)),
			pauseTotalNs.Observation(int64(memStats.PauseTotalNs)),
		)

		computeGCPauses(ctx, &gcPauseNs, memStats.PauseNs[:], lastNumGC, memStats.NumGC)

		lastNumGC = memStats.NumGC
	})

	if heapAlloc, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.heap_alloc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes of allocated heap objects"),
	); err != nil {
		return err
	}

	if heapIdle, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.heap_idle",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes in idle (unused) spans"),
	); err != nil {
		return err
	}

	if heapInuse, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.heap_inuse",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes in in-use spans"),
	); err != nil {
		return err
	}

	if heapObjects, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.heap_objects",
		metric.WithDescription("Number of allocated heap objects"),
	); err != nil {
		return err
	}

	// FYI see https://github.com/golang/go/issues/32284 to help
	// understand the meaning of this value.
	if heapReleased, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.heap_released",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes of idle spans whose physical memory has been returned to the OS"),
	); err != nil {
		return err
	}

	if heapSys, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.heap_sys",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes of heap memory obtained from the OS"),
	); err != nil {
		return err
	}

	if ptrLookups, err = batchObserver.NewInt64CounterObserver(
		"runtime.go.mem.lookups",
		metric.WithDescription("Number of pointer lookups performed by the runtime"),
	); err != nil {
		return err
	}

	if liveObjects, err = batchObserver.NewInt64UpDownCounterObserver(
		"runtime.go.mem.live_objects",
		metric.WithDescription("Number of live objects is the number of cumulative Mallocs - Frees"),
	); err != nil {
		return err
	}

	if gcCount, err = batchObserver.NewInt64CounterObserver(
		"runtime.go.gc.count",
		metric.WithDescription("Number of completed garbage collection cycles"),
	); err != nil {
		return err
	}

	// Note that the following could be derived as a sum of
	// individual pauses, but we may lose individual pauses if the
	// observation interval is too slow.
	if pauseTotalNs, err = batchObserver.NewInt64CounterObserver(
		"runtime.go.gc.pause_total_ns",
		// TODO: nanoseconds units
		metric.WithDescription("Cumulative nanoseconds in GC stop-the-world pauses since the program started"),
	); err != nil {
		return err
	}

	if gcPauseNs, err = r.meter.NewInt64Histogram(
		"runtime.go.gc.pause_ns",
		// TODO: nanoseconds units
		metric.WithDescription("Amount of nanoseconds in GC stop-the-world pauses"),
	); err != nil {
		return err
	}

	return nil
}

func computeGCPauses(
	ctx context.Context,
	recorder *metric.Int64Histogram,
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
	recorder *metric.Int64Histogram,
	pauses []uint64,
) {
	for _, pause := range pauses {
		recorder.Record(ctx, int64(pause))
	}
}

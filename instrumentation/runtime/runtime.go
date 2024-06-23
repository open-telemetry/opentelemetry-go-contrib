// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"context"
	"math"
	"runtime/metrics"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/contrib/instrumentation/runtime/internal/deprecatedruntime"
	"go.opentelemetry.io/contrib/instrumentation/runtime/internal/x"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/runtime"

const (
	goTotalMemory       = "/memory/classes/total:bytes"
	goMemoryReleased    = "/memory/classes/heap/released:bytes"
	goHeapMemory        = "/memory/classes/heap/stacks:bytes"
	goMemoryLimit       = "/gc/gomemlimit:bytes"
	goMemoryAllocated   = "/gc/heap/allocs:bytes"
	goMemoryAllocations = "/gc/heap/allocs:objects"
	goMemoryGoal        = "/gc/heap/goal:bytes"
	goGoroutines        = "/sched/goroutines:goroutines"
	goMaxProcs          = "/sched/gomaxprocs:threads"
	goConfigGC          = "/gc/gogc:percent"
)

// Start initializes reporting of runtime metrics using the supplied config.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	if c.MinimumReadMemStatsInterval < 0 {
		c.MinimumReadMemStatsInterval = DefaultMinimumReadMemStatsInterval
	}
	if c.MeterProvider == nil {
		c.MeterProvider = otel.GetMeterProvider()
	}
	meter := c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
	)
	if x.DeprecatedRuntimeMetrics.Enabled() {
		return deprecatedruntime.Start(meter, c.MinimumReadMemStatsInterval)
	}
	memoryUsedInstrument, err := meter.Int64ObservableUpDownCounter(
		"go.memory.used",
		metric.WithUnit("By"),
		metric.WithDescription("Memory used by the Go runtime."),
	)
	if err != nil {
		return err
	}
	memoryLimitInstrument, err := meter.Int64ObservableUpDownCounter(
		"go.memory.limit",
		metric.WithUnit("By"),
		metric.WithDescription("Go runtime memory limit configured by the user, if a limit exists."),
	)
	if err != nil {
		return err
	}
	memoryAllocatedInstrument, err := meter.Int64ObservableCounter(
		"go.memory.allocated",
		metric.WithUnit("By"),
		metric.WithDescription("Memory allocated to the heap by the application."),
	)
	if err != nil {
		return err
	}
	memoryAllocationsInstrument, err := meter.Int64ObservableCounter(
		"go.memory.allocations",
		metric.WithUnit("{allocation}"),
		metric.WithDescription("Count of allocations to the heap by the application."),
	)
	if err != nil {
		return err
	}
	memoryGCGoalInstrument, err := meter.Int64ObservableUpDownCounter(
		"go.memory.gc.goal",
		metric.WithUnit("By"),
		metric.WithDescription("Heap size target for the end of the GC cycle."),
	)
	if err != nil {
		return err
	}
	goroutineCountInstrument, err := meter.Int64ObservableUpDownCounter(
		"go.goroutine.count",
		metric.WithUnit("{goroutine}"),
		metric.WithDescription("Count of live goroutines."),
	)
	if err != nil {
		return err
	}
	processorLimitInstrument, err := meter.Int64ObservableUpDownCounter(
		"go.processor.limit",
		metric.WithUnit("{thread}"),
		metric.WithDescription("The number of OS threads that can execute user-level Go code simultaneously."),
	)
	if err != nil {
		return err
	}
	gogcConfigInstrument, err := meter.Int64ObservableUpDownCounter(
		"go.config.gogc",
		metric.WithUnit("%"),
		metric.WithDescription("Heap size target percentage configured by the user, otherwise 100."),
	)
	if err != nil {
		return err
	}

	otherMemoryOpt := metric.WithAttributeSet(
		attribute.NewSet(attribute.String("go.memory.type", "other")),
	)
	stackMemoryOpt := metric.WithAttributeSet(
		attribute.NewSet(attribute.String("go.memory.type", "stack")),
	)
	collector := newCollector(c.MinimumReadMemStatsInterval)
	var lock sync.Mutex
	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			lock.Lock()
			defer lock.Unlock()
			collector.refresh()
			stackMemory := collector.get(goHeapMemory)
			o.ObserveInt64(memoryUsedInstrument, stackMemory, stackMemoryOpt)
			totalMemory := collector.get(goTotalMemory) - collector.get(goMemoryReleased)
			otherMemory := totalMemory - stackMemory
			o.ObserveInt64(memoryUsedInstrument, otherMemory, otherMemoryOpt)
			// Only observe the limit metric if a limit exists
			if limit := collector.get(goMemoryLimit); limit != math.MaxInt64 {
				o.ObserveInt64(memoryLimitInstrument, limit)
			}
			o.ObserveInt64(memoryAllocatedInstrument, collector.get(goMemoryAllocated))
			o.ObserveInt64(memoryAllocationsInstrument, collector.get(goMemoryAllocations))
			o.ObserveInt64(memoryGCGoalInstrument, collector.get(goMemoryGoal))
			o.ObserveInt64(goroutineCountInstrument, collector.get(goGoroutines))
			o.ObserveInt64(processorLimitInstrument, collector.get(goMaxProcs))
			o.ObserveInt64(gogcConfigInstrument, collector.get(goConfigGC))
			return nil
		},
		memoryUsedInstrument,
		memoryLimitInstrument,
		memoryAllocatedInstrument,
		memoryAllocationsInstrument,
		memoryGCGoalInstrument,
		goroutineCountInstrument,
		processorLimitInstrument,
		gogcConfigInstrument,
	)
	if err != nil {
		return err
	}
	// TODO (#5655) support go.schedule.duration
	return nil
}

// These are the metrics we actually fetch from the go runtime.
var runtimeMetrics = []string{
	goTotalMemory,
	goMemoryReleased,
	goHeapMemory,
	goMemoryLimit,
	goMemoryAllocated,
	goMemoryAllocations,
	goMemoryGoal,
	goGoroutines,
	goMaxProcs,
	goConfigGC,
}

type goCollector struct {
	// now is used to replace the implementation of time.Now for testing
	now func() time.Time
	// lastCollect tracks the last time metrics were refreshed
	lastCollect time.Time
	// minimumInterval is the minimum amount of time between calls to metrics.Read
	minimumInterval time.Duration
	// sampleBuffer is populated by runtime/metrics
	sampleBuffer []metrics.Sample
	// sampleMap allows us to easily get the value of a single metric
	sampleMap map[string]*metrics.Sample
}

func newCollector(minimumInterval time.Duration) *goCollector {
	g := &goCollector{
		sampleBuffer:    make([]metrics.Sample, 0, len(runtimeMetrics)),
		sampleMap:       make(map[string]*metrics.Sample, len(runtimeMetrics)),
		minimumInterval: minimumInterval,
		now:             time.Now,
	}
	for _, runtimeMetric := range runtimeMetrics {
		g.sampleBuffer = append(g.sampleBuffer, metrics.Sample{Name: runtimeMetric})
		g.sampleMap[runtimeMetric] = &g.sampleBuffer[len(g.sampleBuffer)-1]
	}
	return g
}

func (g *goCollector) refresh() {
	now := g.now()
	if now.Sub(g.lastCollect) < g.minimumInterval {
		// refresh was invoked more frequently than allowed by the minimum
		// interval. Do nothing.
		return
	}
	metrics.Read(g.sampleBuffer)
	g.lastCollect = now
}

func (g *goCollector) get(name string) int64 {
	if s, ok := g.sampleMap[name]; ok && s.Value.Kind() == metrics.KindUint64 {
		return int64(s.Value.Uint64())
	}
	return 0
}

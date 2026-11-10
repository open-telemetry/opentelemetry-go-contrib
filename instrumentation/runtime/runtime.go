// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"math"
	"runtime/metrics"
	"slices"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricx "go.opentelemetry.io/otel/metric/x"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/semconv/v1.43.0/goconv"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/runtime"

const (
	goHeapMemory                = "/memory/classes/heap/stacks:bytes"
	goMemoryHeapFree            = "/memory/classes/heap/free:bytes"
	goMemoryHeapObjects         = "/memory/classes/heap/objects:bytes"
	goMemoryHeapUnused          = "/memory/classes/heap/unused:bytes"
	goMemoryMetadataMCacheFree  = "/memory/classes/metadata/mcache/free:bytes"
	goMemoryMetadataMCacheInuse = "/memory/classes/metadata/mcache/inuse:bytes"
	goMemoryMetadataMSpanFree   = "/memory/classes/metadata/mspan/free:bytes"
	goMemoryMetadataMSpanInuse  = "/memory/classes/metadata/mspan/inuse:bytes"
	goMemoryMetadataOther       = "/memory/classes/metadata/other:bytes"
	goMemoryOsStacks            = "/memory/classes/os-stacks:bytes"
	goMemoryOther               = "/memory/classes/other:bytes"
	goMemoryProfilingBuckets    = "/memory/classes/profiling/buckets:bytes"
	goMemoryLimit               = "/gc/gomemlimit:bytes"
	goMemoryAllocated           = "/gc/heap/allocs:bytes"
	goMemoryAllocations         = "/gc/heap/allocs:objects"
	goMemoryGoal                = "/gc/heap/goal:bytes"
	goGoroutines                = "/sched/goroutines:goroutines"
	goMaxProcs                  = "/sched/gomaxprocs:threads"
	goConfigGC                  = "/gc/gogc:percent"
	goSchedLatencies            = "/sched/latencies:seconds"
	goGCCycles                  = "/gc/cycles/total:gc-cycles"
	goGCPauses                  = "/sched/pauses/total/gc:seconds"
	goCPUUser                   = "/cpu/classes/user:cpu-seconds"
	goCPUGCMarkAssist           = "/cpu/classes/gc/mark/assist:cpu-seconds"
	goCPUGCMarkDedicated        = "/cpu/classes/gc/mark/dedicated:cpu-seconds"
	goCPUGCMarkIdle             = "/cpu/classes/gc/mark/idle:cpu-seconds"
	goCPUGCPause                = "/cpu/classes/gc/pause:cpu-seconds"
	goCPUScavengeAssist         = "/cpu/classes/scavenge/assist:cpu-seconds"
	goCPUScavengeBackground     = "/cpu/classes/scavenge/background:cpu-seconds"
	goCPUIdle                   = "/cpu/classes/idle:cpu-seconds"
)

// Start initializes reporting of runtime metrics using the supplied config.
// For goroutine scheduling and GC pause metrics, additionally see [NewProducer].
//
// Metrics emitted by Start includes:
//
//	go.memory.used          By            Memory used by the Go runtime.
//	go.memory.limit         By            Go runtime memory limit configured by the user, if a limit exists.
//	go.memory.allocated     By            Memory allocated to the heap by the application.
//	go.memory.allocations   {allocation}  Count of allocations to the heap by the application.
//	go.memory.gc.goal       By            Heap size target for the end of the GC cycle.
//	go.goroutine.count      {goroutine}   Count of live goroutines.
//	go.processor.limit      {thread}      The number of OS threads that can execute user-level Go code simultaneously.
//	go.config.gogc          %             Heap size target percentage configured by the user, otherwise 100.
//
// The following opt-in metrics are produced when listed, comma-separated, in
// the OTEL_GO_X_RUNTIME_METRICS_OPTIN environment variable:
//
//	go.memory.gc.cycles     {gc_cycle}    Number of completed GC cycles.
//	go.cpu.time             s             Estimated CPU time spent by the Go runtime.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	meter := c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
		metric.WithSchemaURL(semconv.SchemaURL),
	)
	memoryUsed, err := goconv.NewMemoryUsed(
		meter,
		metricx.WithDefaultAttributes((goconv.MemoryUsed{}).AttrMemoryType("").Key),
	)
	if err != nil {
		return err
	}
	memoryLimit, err := goconv.NewMemoryLimit(meter)
	if err != nil {
		return err
	}
	memoryAllocated, err := goconv.NewMemoryAllocated(meter)
	if err != nil {
		return err
	}
	memoryAllocations, err := goconv.NewMemoryAllocations(meter)
	if err != nil {
		return err
	}
	memoryGCGoal, err := goconv.NewMemoryGCGoal(meter)
	if err != nil {
		return err
	}
	goroutineCount, err := goconv.NewGoroutineCount(meter)
	if err != nil {
		return err
	}
	processorLimit, err := goconv.NewProcessorLimit(meter)
	if err != nil {
		return err
	}
	configGogc, err := goconv.NewConfigGogc(meter)
	if err != nil {
		return err
	}

	memoryClasses := newMemoryClasses(memoryUsed)
	instruments := []metric.Observable{
		memoryUsed.Inst(),
		memoryLimit.Inst(),
		memoryAllocated.Inst(),
		memoryAllocations.Inst(),
		memoryGCGoal.Inst(),
		goroutineCount.Inst(),
		processorLimit.Inst(),
		configGogc.Inst(),
	}
	metricNames := slices.Clone(runtimeMetrics)

	enableGCCycles := optInGCCycles.enabled()
	var gcCycles goconv.MemoryGCCyclesObservable
	if enableGCCycles {
		gcCycles, err = goconv.NewMemoryGCCyclesObservable(meter)
		if err != nil {
			return err
		}
		instruments = append(instruments, gcCycles.Inst())
		metricNames = append(metricNames, goGCCycles)
	}

	enableCPUTime := optInCPUTime.enabled()
	var cpuTime goconv.CPUTimeObservable
	var cpuStates []cpuState
	if enableCPUTime {
		cpuTime, err = goconv.NewCPUTimeObservable(
			meter,
			metricx.WithDefaultAttributes((goconv.CPUTimeObservable{}).AttrCPUState("").Key),
		)
		if err != nil {
			return err
		}
		cpuStates = newCPUStates(cpuTime)
		instruments = append(instruments, cpuTime.Inst())
		for _, state := range cpuStates {
			metricNames = append(metricNames, state.runtimeMetric)
		}
	}

	collector := newCollector(c.MinimumRefreshInterval, metricNames)
	var lock sync.Mutex
	_, err = meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			lock.Lock()
			defer lock.Unlock()
			collector.refresh()
			for _, memClass := range memoryClasses {
				o.ObserveInt64(memoryUsed.Inst(), collector.getInt(memClass.runtimeMetric), memClass.attributes)
			}
			// Only observe the limit metric if a limit exists
			if limit := collector.getInt(goMemoryLimit); limit != math.MaxInt64 {
				o.ObserveInt64(memoryLimit.Inst(), limit)
			}
			o.ObserveInt64(memoryAllocated.Inst(), collector.getInt(goMemoryAllocated))
			o.ObserveInt64(memoryAllocations.Inst(), collector.getInt(goMemoryAllocations))
			o.ObserveInt64(memoryGCGoal.Inst(), collector.getInt(goMemoryGoal))
			o.ObserveInt64(goroutineCount.Inst(), collector.getInt(goGoroutines))
			o.ObserveInt64(processorLimit.Inst(), collector.getInt(goMaxProcs))
			o.ObserveInt64(configGogc.Inst(), collector.getInt(goConfigGC))
			if enableGCCycles {
				o.ObserveInt64(gcCycles.Inst(), collector.getInt(goGCCycles))
			}
			for _, state := range cpuStates {
				o.ObserveFloat64(cpuTime.Inst(), collector.getFloat(state.runtimeMetric), state.attributes)
			}
			return nil
		},
		instruments...,
	)
	if err != nil {
		return err
	}
	return nil
}

// These are the metrics we actually fetch from the go runtime.
var runtimeMetrics = []string{
	goHeapMemory,
	goMemoryHeapFree,
	goMemoryHeapObjects,
	goMemoryHeapUnused,
	goMemoryMetadataMCacheFree,
	goMemoryMetadataMCacheInuse,
	goMemoryMetadataMSpanFree,
	goMemoryMetadataMSpanInuse,
	goMemoryMetadataOther,
	goMemoryOsStacks,
	goMemoryOther,
	goMemoryProfilingBuckets,
	goMemoryLimit,
	goMemoryAllocated,
	goMemoryAllocations,
	goMemoryGoal,
	goGoroutines,
	goMaxProcs,
	goConfigGC,
}

// memoryClass maps a runtime memory class to its go.memory.type and
// go.memory.detailed_type attributes.
type memoryClass struct {
	runtimeMetric string
	attributes    metric.ObserveOption
}

func newMemoryClasses(memoryUsed goconv.MemoryUsed) []memoryClass {
	newClass := func(runtimeMetric string, mType goconv.MemoryTypeAttr, detailedType string) memoryClass {
		return memoryClass{
			runtimeMetric: runtimeMetric,
			attributes: metric.WithAttributeSet(attribute.NewSet(
				memoryUsed.AttrMemoryType(mType),
				memoryUsed.AttrMemoryDetailedType(detailedType),
			)),
		}
	}
	return []memoryClass{
		newClass(goHeapMemory, goconv.MemoryTypeStack, "heap/stacks"),
		newClass(goMemoryHeapFree, goconv.MemoryTypeOther, "heap/free"),
		newClass(goMemoryHeapObjects, goconv.MemoryTypeOther, "heap/objects"),
		newClass(goMemoryHeapUnused, goconv.MemoryTypeOther, "heap/unused"),
		newClass(goMemoryMetadataMCacheFree, goconv.MemoryTypeOther, "metadata/mcache/free"),
		newClass(goMemoryMetadataMCacheInuse, goconv.MemoryTypeOther, "metadata/mcache/inuse"),
		newClass(goMemoryMetadataMSpanFree, goconv.MemoryTypeOther, "metadata/mspan/free"),
		newClass(goMemoryMetadataMSpanInuse, goconv.MemoryTypeOther, "metadata/mspan/inuse"),
		newClass(goMemoryMetadataOther, goconv.MemoryTypeOther, "metadata/other"),
		newClass(goMemoryOsStacks, goconv.MemoryTypeOther, "os-stacks"),
		newClass(goMemoryOther, goconv.MemoryTypeOther, "other"),
		newClass(goMemoryProfilingBuckets, goconv.MemoryTypeOther, "profiling/buckets"),
	}
}

// cpuState maps a runtime CPU class to its go.cpu.state and
// go.cpu.detailed_state attributes.
type cpuState struct {
	runtimeMetric string
	attributes    metric.ObserveOption
}

func newCPUStates(cpuTime goconv.CPUTimeObservable) []cpuState {
	newState := func(runtimeMetric string, state goconv.CPUStateAttr, detailedState string) cpuState {
		return cpuState{
			runtimeMetric: runtimeMetric,
			attributes: metric.WithAttributeSet(attribute.NewSet(
				cpuTime.AttrCPUState(state),
				cpuTime.AttrCPUDetailedState(detailedState),
			)),
		}
	}
	return []cpuState{
		newState(goCPUUser, goconv.CPUStateUser, "user"),
		newState(goCPUGCMarkAssist, goconv.CPUStateGC, "gc/mark/assist"),
		newState(goCPUGCMarkDedicated, goconv.CPUStateGC, "gc/mark/dedicated"),
		newState(goCPUGCMarkIdle, goconv.CPUStateGC, "gc/mark/idle"),
		newState(goCPUGCPause, goconv.CPUStateGC, "gc/pause"),
		newState(goCPUScavengeAssist, goconv.CPUStateScavenge, "scavenge/assist"),
		newState(goCPUScavengeBackground, goconv.CPUStateScavenge, "scavenge/background"),
		newState(goCPUIdle, goconv.CPUStateIdle, "idle"),
	}
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

func newCollector(minimumInterval time.Duration, metricNames []string) *goCollector {
	g := &goCollector{
		sampleBuffer:    make([]metrics.Sample, 0, len(metricNames)),
		sampleMap:       make(map[string]*metrics.Sample, len(metricNames)),
		minimumInterval: minimumInterval,
		now:             time.Now,
	}
	for _, metricName := range metricNames {
		g.sampleBuffer = append(g.sampleBuffer, metrics.Sample{Name: metricName})
		// sampleMap references a position in the sampleBuffer slice. If an
		// element is appended to sampleBuffer, it must be added to sampleMap
		// for the sample to be accessible in sampleMap.
		g.sampleMap[metricName] = &g.sampleBuffer[len(g.sampleBuffer)-1]
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

func (g *goCollector) getInt(name string) int64 {
	if s, ok := g.sampleMap[name]; ok && s.Value.Kind() == metrics.KindUint64 {
		v := s.Value.Uint64()
		if v > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(v)
	}
	return 0
}

func (g *goCollector) getFloat(name string) float64 {
	if s, ok := g.sampleMap[name]; ok && s.Value.Kind() == metrics.KindFloat64 {
		return s.Value.Float64()
	}
	return 0
}

func (g *goCollector) getHistogram(name string) *metrics.Float64Histogram {
	if s, ok := g.sampleMap[name]; ok && s.Value.Kind() == metrics.KindFloat64Histogram {
		return s.Value.Float64Histogram()
	}
	return nil
}

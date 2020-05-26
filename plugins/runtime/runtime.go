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

package runtime // import "go.opentelemetry.io/contrib/plugins/runtime"

// issues induced by this file:
// want nanosecond units in otel/api/unit

import (
	"context"
	"errors"
	goruntime "runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
)

// Runtime reports the work-in-progress conventional runtime metrics specified by OpenTelemetry
type Runtime struct {
	mu       sync.RWMutex
	meter    metric.Meter
	interval time.Duration
	batchObs metric.BatchObserver

	instruments struct {
		// Runtime
		goCgoCalls metric.Int64SumObserver

		// Memstats
		goPtrLookups metric.Int64SumObserver

		// GC stats
		gcCount   metric.Int64SumObserver
		gcPauseNs metric.Int64ValueRecorder
	}

	cacheMemStats goruntime.MemStats
}

// New returns Runtime, a structure for reporting Go runtime metrics
// interval is used to define how often to invoke Go runtime.ReadMemStats() to obtain metric data. It should be noted
// this package invokes a stop-the-world function on this interval. The interval should not be set arbitrarily small
// without accepting the performance overhead.
//
// TODO this interval may be removed in favor of otel SDK control after batch observers land
func New(meter metric.Meter, interval time.Duration) *Runtime {
	r := &Runtime{
		meter:    meter,
		interval: interval,
	}

	return r
}

// Start begins regular background polling of Go runtime metrics and will return an error if any issues are encountered
func (r *Runtime) Start() error {
	if r.interval <= 0 {
		return errors.New("non-positive interval for runtime.New")
	}

	err := r.register()
	if err != nil {
		return err
	}

	return nil
}

func (r *Runtime) register() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.batchObs = r.meter.NewBatchObserver(r.collect)

	var err error

	t0 := time.Now()
	if _, err = r.meter.NewInt64SumObserver(
		"runtime.uptime",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(time.Since(t0).Milliseconds())
		},
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("Milliseconds since application was initialized"),
	); err != nil {
		return err
	}

	if _, err = r.meter.NewInt64SumObserver(
		"runtime.go.goroutines",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(int64(goruntime.NumGoroutine()))
		},
		metric.WithDescription("Number of goroutines that currently exist"),
	); err != nil {
		return err
	}

	if r.instruments.goCgoCalls, err = r.batchObs.NewInt64SumObserver(
		"runtime.go.cgo.calls",
		metric.WithDescription("Number of cgo calls made by the current process"),
	); err != nil {
		return err
	}

	err = r.registerMemStats()
	if err != nil {
		return err
	}

	err = r.registerGcStats()
	if err != nil {
		return err
	}

	// TODO go version as tag: make this a sub-package for providing standard runtime labels?

	return nil
}

func (r *Runtime) registerMemStats() error {
	var (
		err error

		heapAlloc metric.Int64UpDownSumObserver
		heapIdle  metric.Int64UpDownSumObserver
	)
	// NOTE @@@ HERE YOU ARE
	// deciding when to use labels depends on subsetting behavior (for the doc) #obviously.
	// :boom: https://github.com/golang/go/issues/32284

	batchObserver := r.meter.NewBatchObserver(func(result metric.BatchObserverResult) {
		result.Observe(nil)
	})

	if heapSize, err = batchObserver.NewInt64SumObserver(
		"runtime.go.mem.heap_alloc",
		// func(_ context.Context, result metric.Int64ObserverResult) {
		// 	r.mu.RLock()
		// 	defer r.mu.RUnlock()
		// 	result.Observe(int64(r.memStats.HeapAlloc))
		// },
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes of allocated heap objects"),
	); err != nil {
		return err
	}

	// if _, err = r.meter.NewInt64SumObserver(
	// 	"runtime.go.mem.heap_idle",
	// 	// func(_ context.Context, result metric.Int64ObserverResult) {
	// 	// 	r.mu.RLock()
	// 	// 	defer r.mu.RUnlock()
	// 	// 	result.Observe(int64(r.memStats.HeapIdle))
	// 	// },
	// 	metric.WithUnit(unit.Bytes),
	// 	metric.WithDescription("Bytes in idle (unused) spans"),
	// ); err != nil {
	// 	return err
	// }

	// if _, err = r.meter.NewInt64SumObserver(
	// 	"runtime.go.mem.heap_inuse",
	// 	func(_ context.Context, result metric.Int64ObserverResult) {
	// 		r.mu.RLock()
	// 		defer r.mu.RUnlock()
	// 		result.Observe(int64(r.memStats.HeapInuse))
	// 	},
	// 	metric.WithUnit(unit.Bytes),
	// 	metric.WithDescription("Bytes in in-use spans"),
	// ); err != nil {
	// 	return err
	// }

	// if _, err = r.meter.NewInt64SumObserver(
	// 	"runtime.go.mem.heap_objects",
	// 	func(_ context.Context, result metric.Int64ObserverResult) {
	// 		r.mu.RLock()
	// 		defer r.mu.RUnlock()
	// 		result.Observe(int64(r.memStats.HeapObjects))
	// 	},
	// 	metric.WithDescription("Number of allocated heap objects"),
	// ); err != nil {
	// 	return err
	// }

	// // https://github.com/golang/go/issues/32284 is actually gauge
	// // Post spec 0.4 -> Int64ValueObserver (?)
	// if _, err = r.meter.NewInt64SumObserver(
	// 	"runtime.go.mem.heap_released",
	// 	func(result metric.Int64ObserverResult) {
	// 		r.mu.RLock()
	// 		defer r.mu.RUnlock()
	// 		result.Observe(int64(r.memStats.HeapReleased))
	// 	},
	// 	metric.WithUnit(unit.Bytes),
	// 	metric.WithDescription("Bytes of idle spans whose physical memory has been returned to the OS"),
	// ); err != nil {
	// 	return err
	// }

	// if _, err = r.meter.NewInt64SumObserver(
	// 	"runtime.go.mem.heap_sys",
	// 	func(_ context.Context, result metric.Int64ObserverResult) {
	// 		r.mu.RLock()
	// 		defer r.mu.RUnlock()
	// 		result.Observe(int64(r.memStats.HeapSys))
	// 	},
	// 	metric.WithUnit(unit.Bytes),
	// 	metric.WithDescription("Bytes of heap memory obtained from the OS"),
	// ); err != nil {
	// 	return err
	// }

	if r.instruments.goPtrLookups, err = r.meter.NewInt64Counter(
		"runtime.go.mem.lookups",
		metric.WithDescription("Number of pointer lookups performed by the runtime"),
	); err != nil {
		return err
	}

	if _, err = r.meter.NewInt64SumObserver(
		"runtime.go.mem.live_objects",
		func(_ context.Context, result metric.Int64ObserverResult) {
			r.mu.RLock()
			defer r.mu.RUnlock()
			result.Observe(int64(r.memStats.Mallocs - r.memStats.Frees))
		},
		metric.WithDescription("Number of live objects is the number of cumulative Mallocs - Frees"),
	); err != nil {
		return err
	}

	return err
}

func (r *Runtime) registerGcStats() error {
	var err error

	r.instruments.gcCount, err = r.meter.NewInt64Counter("runtime.go.gc.count",
		metric.WithDescription("Number of completed garbage collection cycles"))
	if err != nil {
		return err
	}

	_, err = r.meter.NewInt64SumObserver("runtime.go.gc.pause_total_ns", func(result metric.Int64ObserverResult) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		result.Observe(int64(r.memStats.PauseTotalNs))
	}, metric.WithDescription("Cumulative nanoseconds in GC stop-the-world pauses since the program started"))
	if err != nil {
		return err
	}

	r.instruments.gcPauseNs, err = r.meter.NewInt64Measure("runtime.go.gc.pause_ns",
		metric.WithDescription("Amount of nanoseconds in GC stop-the-world pauses"))
	if err != nil {
		return err
	}

	return nil
}

func (r *Runtime) collect(ctx context.Context, result metric.BatchObserverResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lastNumCgoCalls := r.numCgoCalls
	r.numCgoCalls = goruntime.NumCgoCall()
	r.instruments.goCgoCalls.Add(ctx, r.numCgoCalls-lastNumCgoCalls)

	lastLookups := r.memStats.Lookups
	lastNumGC := r.memStats.NumGC

	pauses := collectMemoryStats(&r.memStats, lastNumGC)

	r.instruments.goPtrLookups.Add(ctx, int64(r.memStats.Lookups-lastLookups))
	r.instruments.gcCount.Add(ctx, int64(r.memStats.NumGC-lastNumGC))

	for _, pause := range pauses {
		r.instruments.gcPauseNs.Record(ctx, pause.Nanoseconds())
	}
}

func collectMemoryStats(memStats *goruntime.MemStats, lastNumGC uint32) (pauses []time.Duration) {
	goruntime.ReadMemStats(memStats)
	return makeGCPauses(memStats, lastNumGC)
}

func makeGCPauses(memStats *goruntime.MemStats, lastNumGC uint32) (pauses []time.Duration) {
	delta := int(memStats.NumGC - lastNumGC)

	if delta == 0 {
		return nil
	}

	if delta >= len(memStats.PauseNs) {
		return makePauses(memStats.PauseNs[:], nil)
	}

	length := uint32(len(memStats.PauseNs))
	offset := length - 1

	i := (lastNumGC + offset + 1) % length
	j := (memStats.NumGC + offset + 1) % length

	if j < i { // wrap around the circular buffer
		return makePauses(memStats.PauseNs[i:], memStats.PauseNs[:j])
	}

	return makePauses(memStats.PauseNs[i:j], nil)
}

func makePauses(head []uint64, tail []uint64) (pauses []time.Duration) {
	pauses = make([]time.Duration, 0, len(head)+len(tail))
	pauses = appendPauses(pauses, head)
	pauses = appendPauses(pauses, tail)
	return
}

func appendPauses(pauses []time.Duration, values []uint64) []time.Duration {
	for _, v := range values {
		pauses = append(pauses, time.Duration(v))
	}
	return pauses
}

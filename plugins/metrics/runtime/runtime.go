package runtime // import "go.opentelemetry.io/contrib/plugins/metrics/runtime"

import (
	"context"
	goruntime "runtime"
	"time"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
)

type Runtime struct {
	meter    metric.Meter
	interval time.Duration
	done     chan bool

	metrics struct {
		goCgoCalls metric.Int64Counter
		goLookups  metric.Int64Counter
		goGcCount  metric.Int64Counter
		gcPauseNs  metric.Int64Measure
	}

	memStats    goruntime.MemStats
	numCgoCalls int64
}

func NewRuntime(meter metric.Meter, interval time.Duration) *Runtime {
	r := &Runtime{
		meter:    meter,
		interval: interval,
		done:     make(chan bool),
	}

	return r
}

func (r *Runtime) Start() error {
	err := r.register()
	if err != nil {
		return err
	}

	go r.ticker()

	return nil
}

func (r *Runtime) Stop() {
	r.done <- true
}

func (r *Runtime) ticker() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			ctx := context.Background()
			r.collect(ctx)
		}
	}
}

func (r *Runtime) register() error {
	var err error

	t0 := time.Now()
	_, err = r.meter.RegisterInt64Observer("runtime.uptime",
		func(result metric.Int64ObserverResult) {
			result.Observe(time.Since(t0).Milliseconds())
		},
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("milliseconds since application was initialized"),
	)
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.goroutines", func(result metric.Int64ObserverResult) {
		result.Observe(int64(goruntime.NumGoroutine()))
	}, metric.WithDescription("Number of goroutines that currently exist"))
	if err != nil {
		return err
	}

	r.metrics.goCgoCalls, err = r.meter.NewInt64Counter("runtime.go.cgo.calls",
		metric.WithDescription("Number of cgo calls made by the current process"))
	if err != nil {
		return err
	}

	// poll now so that the first tick has a full delta
	r.numCgoCalls = goruntime.NumCgoCall()
	goruntime.ReadMemStats(&r.memStats)

	err = r.registerMemStats()
	if err != nil {
		return err
	}

	err = r.registerGcStats()
	if err != nil {
		return err
	}

	// TODO go version as tag

	return nil
}

func (r *Runtime) registerMemStats() error {
	var err error

	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.heap_alloc", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.HeapAlloc))
	}, metric.WithUnit(unit.Bytes), metric.WithDescription("Bytes of allocated heap objects"))
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.heap_idle", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.HeapIdle))
	}, metric.WithUnit(unit.Bytes), metric.WithDescription("Bytes in idle (unused) spans"))
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.heap_inuse", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.HeapInuse))
	}, metric.WithUnit(unit.Bytes), metric.WithDescription("Bytes in in-use spans"))
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.heap_objects", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.HeapObjects))
	}, metric.WithDescription("Number of allocated heap objects"))
	if err != nil {
		return err
	}

	// https://github.com/golang/go/issues/32284 is actually gauge
	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.heap_released", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.HeapReleased))
	}, metric.WithUnit(unit.Bytes),
		metric.WithDescription("Bytes of idle spans whose physical memory has been returned to the OS"))
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.heap_sys", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.HeapSys))
	}, metric.WithUnit(unit.Bytes), metric.WithDescription("Bytes of heap memory obtained from the OS"))
	if err != nil {
		return err
	}

	r.metrics.goLookups, err = r.meter.NewInt64Counter("runtime.go.lookups",
		metric.WithDescription("Number of pointer lookups performed by the runtime"))
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.mem.live_objects", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.Mallocs - r.memStats.Frees))
	}, metric.WithDescription("Number of live objects is the number of cumulative Mallocs - Frees"))
	if err != nil {
		return err
	}

	return err
}

func (r *Runtime) registerGcStats() error {
	var err error

	r.metrics.goGcCount, err = r.meter.NewInt64Counter("runtime.go.gc.count",
		metric.WithDescription("Number of completed garbage collection cycles"))
	if err != nil {
		return err
	}

	_, err = r.meter.RegisterInt64Observer("runtime.go.gc.pause_total_ns", func(result metric.Int64ObserverResult) {
		result.Observe(int64(r.memStats.PauseTotalNs))
	}, metric.WithDescription("Cumulative nanoseconds in GC stop-the-world pauses since the program started"))
	if err != nil {
		return err
	}

	r.metrics.gcPauseNs, err = r.meter.NewInt64Measure("runtime.go.gc.pause_ns",
		metric.WithDescription("Amount of nanoseconds in GC stop-the-world pauses"))
	if err != nil {
		return err
	}

	return nil
}

func (r *Runtime) collect(ctx context.Context) {
	lastNumCgoCalls := r.numCgoCalls
	r.numCgoCalls = goruntime.NumCgoCall()
	r.metrics.goCgoCalls.Add(ctx, r.numCgoCalls-lastNumCgoCalls)

	lastLookups := r.memStats.Lookups
	lastNumGC := r.memStats.NumGC

	pauses := collectMemoryStats(&r.memStats, lastNumGC)

	r.metrics.goLookups.Add(ctx, int64(r.memStats.Lookups-lastLookups))
	r.metrics.goGcCount.Add(ctx, int64(r.memStats.NumGC-lastNumGC))

	for _, pause := range pauses {
		r.metrics.gcPauseNs.Record(ctx, pause.Nanoseconds())
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

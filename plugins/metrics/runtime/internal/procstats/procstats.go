package procstats

import (
	"math"
	"runtime"
	"time"

	// "github.com/segmentio/stats/v4"
)

func init() {
	stats.Buckets.Set("go.memstats:gc_pause.seconds",
		1*time.Microsecond,
		10*time.Microsecond,
		100*time.Microsecond,
		1*time.Millisecond,
		10*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
		math.Inf(+1),
	)
}

// GoMetrics is a metric collector that reports metrics from the Go runtime.
type GoMetrics struct {
	engine  *stats.Engine
	version string `tag:"version"`

	runtime struct {
		// Runtime info.
		numCPU         int `metric:"cpu.num"       type:"gauge"`
		numGoroutine   int `metric:"goroutine.num" type:"gauge"`
		numCgoCall     int `metric:"cgo.calls"     type:"counter"`
		lastNumCgoCall int
	} `metric:"go.runtime"`

	memstats struct {
		// General statistics.
		total struct {
			alloc      uint64 `metric:"alloc.bytes"       type:"gauge"`   // bytes allocated (even if freed)
			totalAlloc uint64 `metric:"total_alloc.bytes" type:"counter"` // bytes allocated (even if freed)
			lookups    uint64 `metric:"lookups.count"     type:"counter"` // number of pointer lookups
			mallocs    uint64 `metric:"mallocs.count"     type:"counter"` // number of mallocs
			frees      uint64 `metric:"frees.count"       type:"counter"` // number of frees
			memtype    string `tag:"type"`
		}

		// Main allocation heap statistics.
		heap struct {
			alloc    uint64 `metric:"alloc.bytes"    type:"gauge"`   // bytes allocated and not yet freed
			sys      uint64 `metric:"sys.bytes"      type:"gauge"`   // bytes obtained from system
			idle     uint64 `metric:"idle.bytes"     type:"gauge"`   // bytes in idle spans
			inuse    uint64 `metric:"inuse.bytes"    type:"gauge"`   // bytes in non-idle span
			released uint64 `metric:"released.bytes" type:"counter"` // bytes released to the OS
			objects  uint64 `metric:"objects.count"  type:"gauge"`   // total number of allocated objects
			memtype  string `tag:"type"`
		}

		// Low-level fixed-size structure allocator statistics.
		stack struct {
			inuse   uint64 `metric:"inuse.bytes" type:"gauge"` // bytes used by stack allocator
			sys     uint64 `metric:"sys.bytes"   type:"gauge"`
			memtype string `tag:"type"`
		}

		mspan struct {
			inuse   uint64 `metric:"inuse.bytes" type:"gauge"` // mspan structures
			sys     uint64 `metric:"sys.bytes"   type:"gauge"`
			memtype string `tag:"type"`
		}

		mcache struct {
			inuse   uint64 `metric:"inuse.bytes" type:"gauge"` // mcache structures
			sys     uint64 `metric:"sys.bytes"   type:"gauge"`
			memtype string `tag:"type"`
		}

		buckhash struct {
			sys     uint64 `metric:"sys.bytes" type:"gauge"` // profiling bucket hash table
			memtype string `tag:"type"`
		}

		gc struct {
			sys     uint64 `metric:"sys.bytes" type:"gauge"` // GC metadata
			memtype string `tag:"type"`
		}

		other struct {
			sys     uint64 `metric:"sys.bytes" type:"gauge"` // other system allocations
			memtype string `tag:"type"`
		}

		// Garbage collector statistics.
		numGC         uint32        `metric:"gc.count"             type:"counter"` // number of garbage collections
		nextGC        uint64        `metric:"gc_next.bytes"        type:"gauge"`   // next collection will happen when HeapAlloc ≥ this amount
		gcPauseAvg    time.Duration `metric:"gc_pause.seconds.avg" type:"gauge"`
		gcPauseMin    time.Duration `metric:"gc_pause.seconds.min" type:"gauge"`
		gcPauseMax    time.Duration `metric:"gc_pause.seconds.max" type:"gauge"`
		gcCPUFraction float64       `metric:"gc_cpu.fraction"      type:"gauge"` // fraction of CPU time used by GC
	} `metric:"go.memstats"`

	// cache
	ms runtime.MemStats
}

// NewGoMetrics creates a new collector for the Go runtime that produces metrics
// on the default stats engine.
func NewGoMetrics() *GoMetrics {
	return NewGoMetricsWith(stats.DefaultEngine)
}

// NewGoMetricsWith creates a new collector for the Go unrtime that producers
// metrics on eng.
func NewGoMetricsWith(eng *stats.Engine) *GoMetrics {
	g := &GoMetrics{
		engine:  eng,
		version: runtime.Version(),
	}

	g.memstats.total.memtype = "total"
	g.memstats.heap.memtype = "heap"
	g.memstats.stack.memtype = "stack"
	g.memstats.mspan.memtype = "mspan"
	g.memstats.mcache.memtype = "mcache"
	g.memstats.buckhash.memtype = "bucket_hash_table"
	g.memstats.gc.memtype = "gc"
	g.memstats.other.memtype = "other"
	return g
}

// Collect satisfies the Collector interface.
func (g *GoMetrics) Collect() {
	now := time.Now()

	lastTotalAlloc := g.ms.TotalAlloc
	lastLookups := g.ms.Lookups
	lastMallocs := g.ms.Mallocs
	lastFrees := g.ms.Frees
	lastHeapRealeased := g.ms.HeapReleased
	lastNumGC := g.ms.NumGC
	lastNumCgoCall := g.runtime.lastNumCgoCall
	g.runtime.lastNumCgoCall = int(runtime.NumCgoCall())

	g.runtime.numCPU = runtime.NumCPU()
	g.runtime.numGoroutine = runtime.NumGoroutine()
	g.runtime.numCgoCall = g.runtime.lastNumCgoCall - lastNumCgoCall

	pauses := collectMemoryStats(&g.ms, lastNumGC)

	g.memstats.total.alloc = g.ms.Alloc
	g.memstats.total.totalAlloc = g.ms.TotalAlloc - lastTotalAlloc
	g.memstats.total.lookups = g.ms.Lookups - lastLookups
	g.memstats.total.mallocs = g.ms.Mallocs - lastMallocs
	g.memstats.total.frees = g.ms.Frees - lastFrees

	g.memstats.heap.alloc = g.ms.HeapAlloc
	g.memstats.heap.sys = g.ms.HeapSys
	g.memstats.heap.idle = g.ms.HeapIdle
	g.memstats.heap.inuse = g.ms.HeapInuse
	g.memstats.heap.released = g.ms.HeapReleased - lastHeapRealeased
	g.memstats.heap.objects = g.ms.HeapObjects

	g.memstats.stack.inuse = g.ms.StackInuse
	g.memstats.stack.sys = g.ms.StackSys
	g.memstats.mspan.inuse = g.ms.MSpanInuse
	g.memstats.mspan.sys = g.ms.MSpanSys
	g.memstats.mcache.inuse = g.ms.MCacheInuse
	g.memstats.mcache.sys = g.ms.MCacheSys
	g.memstats.buckhash.sys = g.ms.BuckHashSys
	g.memstats.gc.sys = g.ms.GCSys
	g.memstats.other.sys = g.ms.OtherSys

	g.memstats.numGC = g.ms.NumGC - lastNumGC
	g.memstats.nextGC = g.ms.NextGC
	g.memstats.gcCPUFraction = g.ms.GCCPUFraction

	if len(pauses) == 0 {
		g.memstats.gcPauseAvg = 0
		g.memstats.gcPauseMin = 0
		g.memstats.gcPauseMax = 0
	} else {
		g.memstats.gcPauseMin = pauses[0]
		g.memstats.gcPauseMax = pauses[0]
		g.memstats.gcPauseAvg = pauses[0]

		for _, pause := range pauses[1:] {
			g.memstats.gcPauseAvg += pause
			switch {
			case pause < g.memstats.gcPauseMin:
				g.memstats.gcPauseMin = pause
			case pause > g.memstats.gcPauseMax:
				g.memstats.gcPauseMax = pause
			}
		}

		g.memstats.gcPauseAvg /= time.Duration(len(pauses))
	}

	g.engine.ReportAt(now, g)
}

func collectMemoryStats(memstats *runtime.MemStats, lastNumGC uint32) (pauses []time.Duration) {
	runtime.ReadMemStats(memstats)
	return makeGCPauses(memstats, lastNumGC)
}

func makeGCPauses(memstats *runtime.MemStats, lastNumGC uint32) (pauses []time.Duration) {
	delta := int(memstats.NumGC - lastNumGC)

	if delta == 0 {
		return nil
	}

	if delta >= len(memstats.PauseNs) {
		return makePauses(memstats.PauseNs[:], nil)
	}

	length := uint32(len(memstats.PauseNs))
	offset := length - 1

	i := (lastNumGC + offset + 1) % length
	j := (memstats.NumGC + offset + 1) % length

	if j < i { // wrap around the circular buffer
		return makePauses(memstats.PauseNs[i:], memstats.PauseNs[:j])
	}

	return makePauses(memstats.PauseNs[i:j], nil)
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

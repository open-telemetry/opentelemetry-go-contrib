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

// package runtime implements two forms of runtime metrics for Golang OpenTelemetry users.
//
// The original implementation in this package uses ReadMemStats()
// directly to report metric names.
//
// The metric events produced are:
//   process.runtime.go.cgo.calls         -          Number of cgo calls made by the current process
//   process.runtime.go.gc.count          -          Number of completed garbage collection cycles
//   process.runtime.go.gc.pause_ns       (ns)       Amount of nanoseconds in GC stop-the-world pauses
//   process.runtime.go.gc.pause_total_ns (ns)       Cumulative nanoseconds in GC stop-the-world pauses since the program started
//   process.runtime.go.goroutines        -          Number of goroutines that currently exist
//   process.runtime.go.lookups           -          Number of pointer lookups performed by the runtime
//   process.runtime.go.mem.heap_alloc    (bytes)    Bytes of allocated heap objects
//   process.runtime.go.mem.heap_idle     (bytes)    Bytes in idle (unused) spans
//   process.runtime.go.mem.heap_inuse    (bytes)    Bytes in in-use spans
//   process.runtime.go.mem.heap_objects  -          Number of allocated heap objects
//   process.runtime.go.mem.heap_released (bytes)    Bytes of idle spans whose physical memory has been returned to the OS
//   process.runtime.go.mem.heap_sys      (bytes)    Bytes of heap memory obtained from the OS
//   process.runtime.go.mem.live_objects  -          Number of live objects is the number of cumulative Mallocs - Frees
//   runtime.uptime                       (ms)       Milliseconds since application was initialized
//
// The Go-1.16 release featured a new runtime/metrics package that gives formal
// metric names to the various quantities.  This package supports the new metrics
// under their Go-specified names by setting `WithUseGoRuntimeMetrics(true)`.
// These metrics are documented at https://pkg.go.dev/runtime/metrics#hdr-Supported_metrics.
//
// The `runtime/metrics` implementation will replace the older
// implementation as the default no sooner than January 2023. The
// older implementation will be removed no sooner than January 2024.
//
// The following metrics are generated in go-1.19.
//
// Name                                                    Unit          Instrument
// ------------------------------------------------------------------------------------
// process.runtime.go.cgo.go-to-c-calls                    {calls}       Counter[int64]
// process.runtime.go.gc.cycles.automatic                  {gc-cycles}   Counter[int64]
// process.runtime.go.gc.cycles.forced                     {gc-cycles}   Counter[int64]
// process.runtime.go.gc.cycles                            {gc-cycles}   Counter[int64]
// process.runtime.go.gc.heap.allocs                       bytes (*)     Counter[int64]
// process.runtime.go.gc.heap.allocs.objects               {objects} (*) Counter[int64]
// process.runtime.go.gc.heap.allocs-by-size               bytes         Histogram[float64] (**)
// process.runtime.go.gc.heap.frees                        bytes (*)     Counter[int64]
// process.runtime.go.gc.heap.frees.objects                {objects} (*) Counter[int64]
// process.runtime.go.gc.heap.frees-by-size                bytes         Histogram[float64] (**)
// process.runtime.go.gc.heap.goal                         bytes         UpDownCounter[int64]
// process.runtime.go.gc.heap.objects                      {objects}     UpDownCounter[int64]
// process.runtime.go.gc.heap.tiny.allocs                  {objects}     Counter[int64]
// process.runtime.go.gc.limiter.last-enabled              {gc-cycle}    UpDownCounter[int64]
// process.runtime.go.gc.pauses                            seconds       Histogram[float64] (**)
// process.runtime.go.gc.stack.starting-size               bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.heap.free             bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.heap.objects          bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.heap.released         bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.heap.stacks           bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.heap.unused           bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.metadata.mcache.free  bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.metadata.mcache.inuse bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.metadata.mspan.free   bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.metadata.mspan.inuse  bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.metadata.other        bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.os-stacks             bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.other                 bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes.profiling.buckets     bytes         UpDownCounter[int64]
// process.runtime.go.memory.classes                       bytes         UpDownCounter[int64]
// process.runtime.go.sched.gomaxprocs                     {threads}     UpDownCounter[int64]
// process.runtime.go.sched.goroutines                     {goroutines}  UpDownCounter[int64]
// process.runtime.go.sched.latencies                      seconds       GaugeHistogram[float64] (**)
//
// (*) Empty unit strings are cases where runtime/metric produces
// duplicate names ignoring the unit string; here we leave the unit in the name
// and set the unit to empty.
// (**) Histograms are not currently implemented, see the related
// issues for an explanation:
// https://github.com/open-telemetry/opentelemetry-specification/issues/2713
// https://github.com/open-telemetry/opentelemetry-specification/issues/2714

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

# OpenTelemetry Go Runtime Metrics

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/instrumentation/runtimemetrics.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/runtimemetrics)

This package provides automatic collection and reporting of Go runtime metrics as OpenTelemetry metrics. It transforms Go's built-in `runtime/metrics` package data into properly named and dimensioned OpenTelemetry instruments with minimal performance overhead.

## Installation

```bash
go get go.opentelemetry.io/contrib/instrumentation/runtimemetrics
```

## Usage

### Basic Usage

To start collecting runtime metrics with default configuration:

```go
import "go.opentelemetry.io/contrib/instrumentation/runtimemetrics"

func main() {
    if err := runtimemetrics.Start(); err != nil {
        log.Fatal(err)
    }
    // Your application code here
}
```

### Custom Configuration

For custom configuration with a specific MeterProvider:

```go
if err := runtimemetrics.Start(
    runtimemetrics.WithMeterProvider(myMeterProvider),
); err != nil {
    log.Fatal(err)
}
```

## Metrics Collected

This instrumentation collects all available Go runtime metrics and transforms them into OpenTelemetry metrics with the `process.runtime.go` prefix. For complete details on available metrics, see the [Go runtime/metrics documentation](https://pkg.go.dev/runtime/metrics). The following kinds of information is included:

- Heap allocation and object counts
- Garbage collection statistics
- Active goroutine counts
- CGo call counts

### Metric Transformations

The package applies several transformations to align Go runtime metrics with OpenTelemetry conventions:

1. **Naming**: All metrics are prefixed with `process.runtime.go` and use dot notation (e.g., `/gc/heap/allocs` becomes `process.runtime.go.gc.heap.allocs`)

2. **Multi-dimensional Metrics**: For metrics with multiple classification levels, a single OpenTelemetry metric is created with multiple attributes. For example, `/cpu/classes/*:cpu-seconds` becomes `process.runtime.go.cpu.time` with attributes like:
   - `class=gc,class2=mark,class3=assist`
   - `class=gc,class2=pause`
   - `class=scavenge,class2=background`
   - `class=idle`
   - `class=user`

3. **Objects/Bytes Counters**: For metrics that track both object counts and bytes, two separate instruments are created: one with the unit (e.g., `process.runtime.go.gc.heap.allocs` in bytes) and one with an `.objects` suffix (e.g., `process.runtime.go.gc.heap.allocs.objects`)

## Version Compatibility

When Go releases a new toolchain with additional runtime metrics, this package will begin to log warnings about unrecognized fields. For example, if runtime metrics named "/gc/scan/*:bytes" appeared without the call `classesUpDownCounter("/gc/scan/*:bytes")` in defs.go, you would see the following at startup:

```text
2025/06/23 14:53:35 unrecognized runtime/metrics name: /gc/scan/globals:bytes
2025/06/23 14:53:35 unrecognized runtime/metrics name: /gc/scan/heap:bytes
2025/06/23 14:53:35 unrecognized runtime/metrics name: /gc/scan/stack:bytes
2025/06/23 14:53:35 unrecognized runtime/metrics name: /gc/scan/total:bytes
```

When this happens, developers are expected to add a statement in defs.go defining the appropriate metric translation. It is safe to use these definitions with older versions of the Go runtime, since they do not define the newer-runtime metrics.

## Performance

This instrumentation is designed for minimal overhead:

- Uses efficient bulk reading via `runtime/metrics.Read()`
- Lazy instrument creation
- Zero-allocation patterns during collection
- No additional goroutines spawned

## Requirements

- Go 1.24.1+
- OpenTelemetry Go SDK v1.36.0+

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `WithMeterProvider` | `metric.MeterProvider` | Custom MeterProvider for creating instruments | `otel.GetMeterProvider()` |

## Note on Histograms

Histogram metrics from the Go runtime are not currently implemented due to OpenTelemetry specification gaps. This decision is documented and tracked for future implementation when the specification provides clearer guidance.

For more details, see the related OpenTelemetry specification issues:

- [Asynchronous Histogram instrument #2713](https://github.com/open-telemetry/opentelemetry-specification/issues/2713)
- [Async histogram instruments should be re-opened for discussion #2714](https://github.com/open-telemetry/opentelemetry-specification/issues/2714)

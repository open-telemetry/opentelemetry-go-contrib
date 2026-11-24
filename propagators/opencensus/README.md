# OpenCensus Binary Propagation Format

## Deprecation

The `go.opentelemetry.io/contrib/propagators/opencensus` module and package are deprecated as of v0.73.0 and will be removed in v0.74.0.
v0.73.0 is the final planned release of this module.
OpenCensus has been [sunset](https://opentelemetry.io/blog/2023/sunsetting-opencensus/), its [compatibility requirements are deprecated in the OpenTelemetry specification](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/compatibility/opencensus.md), and `go.opentelemetry.io/otel/bridge/opencensus` (which this module depends on) is deprecated as of v1.48.0.
Use native OpenTelemetry propagators instead; see [Migrating from OpenCensus](https://opentelemetry.io/docs/specs/otel/compatibility/opencensus/#migration-path).

## The Problem

The [ocgrpc](https://github.com/census-instrumentation/opencensus-go/tree/master/plugin/ocgrpc) GRPC plugin for OpenCensus is hard-coded to use a [Binary propagation format](https://github.com/census-instrumentation/opencensus-go/blob/380f4078db9f3ee20e26a08105ceecccddf872b8/trace/propagation/propagation.go).

A GRPC client and server that use OpenCensus cannot easily migrate to OpenTelemetry because there will be a period of time during which one will use OpenCensus and the other will use OpenTelemetry.  If both client and server export spans to the same trace backend, the server spans will not be a child of the client spans, because they are using different propagation formats.  To be able to easily migrate from OpenCensus to OpenTelemetry, it is necessary to use the OpenCensus binary propagation format with OpenTelemetry.

## Usage

To add the binary propagation format with otelgrpc, use the WithPropagators option to the otelgrpc Interceptors:

```golang
import "go.opentelemetry.io/contrib/propagators/opencensus"

opt := otelgrpc.WithPropagators(opencensus.Binary{})
```

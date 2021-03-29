# OpenCensus Binary Propagation Format

## The Problem

The [ocgrpc](https://github.com/census-instrumentation/opencensus-go/tree/master/plugin/ocgrpc) GRPC plugin for OpenCensus is hard-coded to use a [Binary propagation format](https://github.com/census-instrumentation/opencensus-go/blob/380f4078db9f3ee20e26a08105ceecccddf872b8/trace/propagation/propagation.go).

A GRPC client and server that use OpenCensus cannot easily migrate to OpenTelemetry because there will be a period of time during which one will use OpenCensus and the other will use OpenTelemetry.  If both client and server export spans to the same trace backend, the server spans will not be a child of the client spans, because they are using different propagation formats.  To be able to easily migrate from OpenCensus to OpenTelemetry, it is necessary to use the OpenCensus binary propagation format with OpenTelemetry.

## Usage

To add the binary propagation format with otelgrpc, use the WithPropagators option to the otelgrpc Interceptors:

```golang
import "go.opentelemetry.io/contrib/propagators/opencensus"

opt := otelgrpc.WithPropagators(opencensus.Binary{})
```

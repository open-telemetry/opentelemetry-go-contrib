# OpenTelemetry Go Initialization Layer

## What it is

The OpenTelemetry Go Initialization Layer is a configuration layer that chooses default values for configuration options that many OpenTelemetry users would ultimately configure manually, allowing for minimal code to quickly instrument with OpenTelemetry. It is intended as the main “getting started” path for newcomers and people without advanced configuration needs.

## What it isn’t

The Go Initialization Layer is not a new Go SDK but rather a complementary wrapper around the existing SDK. Certain functionality like resource detectors, autoprop packages, and autoexporter packages, can and should be used from the core SDK to make the configuration layer more resilient to - and compatible with - potential upstream changes. Other functionality that differs from the Go SDK should not be introduced to the Initialization Layer, either by new environment variables or code configuration, including but not limited to changing out processors, adding specific sampling methods, and enabling or disabling specific instrumentation. New functionality may be provided using a separate package, such as vendor-specific requirements.

The Go Initialization Layer is also not intended to be a lightweight configuration option for advanced users who specifically require minimal dependencies and smaller package size. To achieve the minimal configuration for developers, some unused dependencies will be included as a necessity; see Tradeoffs below.

The Go Initialization Layer is ideally used with environment variables; as such, not all configuration options may be available to set directly within code. If there is more desire to use configuration options directly within code, either the regular SDK can be used, or a vendor package can be used that allows for extra configuration options.

## Motivation

The goal of this Initialization Layer is to help users that aren't familiar with OpenTelemetry quickly ramp up on what they need to get going and instrument their applications. [There is currently a lot of boilerplate code required to use OpenTelemetry](https://opentelemetry.io/docs/instrumentation/go/manual/#initializing-a-new-tracer), and users must make decisions early on before instrumentation works - for example, they must determine and find the appropriate exporters, set resource attributes, etc. This Initialization Layer is intended to simplify the setup and minimize decision-making overhead to be able to quickly get started.

Separately, vendors may choose to provide a vendor-specific package compatible with this Initialization Layer for easier exporting to their own backend. Neither the Initialization Layer nor the vendor-specific package are required to use OpenTelemetry, and using the Initialization Layer does not require the end user to use a vendor-specific package.

## Explanation

The Go Initialization Layer provides a single function to reduce the amount of code that needs to be written to get started, and provides additional, straightforward options to add in that still make additional configuration easy.

The vendor-neutral Initialization Layer will live in the opentelemetry-go-contrib repo, and vendors may create separately hosted vendor-specific configuration packages that can be used with the Initialization Layer.

### Getting started

```shell
go get go.opentelemetry.io/contrib/otelinit
```

### Configure

Minimal setup - by default will send all telemetry to `localhost:4317`, with the ability to set environment variables for endpoint, headers, resource attributes, and more as listed in the configuration options noted below:

```go
import "go.opentelemetry.io/contrib/otelinit"

func main() {
    init, err := otelinit.ConfigureOpentelemetry()
    defer init.Shutdown()
}
```

Alternatively, set environment variables or set variables directly in code to override defaults:

```go
import "go.opentelemetry.io/contrib/otelinit"

func main() {
    init, err := otelinit.ConfigureOpentelemetry(
        otelinit.WithTracesExporter(new trace.SpanExporter("jaeger")) // if using non-default exporter
        otelinit.WithPropagators(b3.New()) // if using non-default propagator
        otelinit.WithServiceName("service-name"),
    )
    defer init.Shutdown()
}
```

### Configuration Options

| Config Option               | Env Variable                        | Required | Default              |
| --------------------------  | ----------------------------------- | -------- | -------------------- |
| WithServiceName             | OTEL_SERVICE_NAME                   | n*       | unknown_service:go   |
| WithServiceVersion          | OTEL_SERVICE_VERSION                | n        | -                    |
| WithTracesExporter          | OTEL_TRACES_EXPORTER                | n        | otlp                 |
| WithLogLevel                | OTEL_LOG_LEVEL                      | n        | info                 |
| WithPropagators             | OTEL_PROPAGATORS                    | n        | tracecontext,baggage |
| WithResourceAttributes      | OTEL_RESOURCE_ATTRIBUTES            | n        | -                    |
| WithMetricsEnabled          | OTEL_METRICS_ENABLED                | n        | true                 |
| WithTracesEnabled           | OTEL_TRACES_ENABLED                 | n        | true                 |

*Service name should be set using the `WithServiceName` configuration option, the `OTEL_SERVICE_NAME` environment variable, or by setting the `service.name` in `OTEL_RESOURCE_ATTRIBUTES`. The default service name is based on the SDK's behavior as it conforms to the specification: `unknown_service`, suffixed with either the process name (where possible) or `go`.

The propagator(s) will be set using the `autoprop` package. `WithPropagators` is an option that sets the default `tracecontext,baggage` propagators if none are set using environment variables.

The exporter(s) will be set using the `autoexport` package. `WithTracesExporter` is an option that sets the default span exporter if none are set using environment variables.

### Using a Vendor-Specific Package

Vendors may create and maintain convenience configuration packages to more easily setup for export to a specific backend. This is commonly done to set a specific endpoint, and set specific metadata or headers needed for telemetry - thus overriding defaults. For example, using a Honeycomb package would look like this:

Minimal setup, which sends to the Honeycomb endpoint and requires `HONEYCOMB_API_KEY` environment variable:

```go
import (
    "go.opentelemetry.io/contrib/otelinit"
    _ "github.com/honeycombio/otel-go/otelinit/honeycomb"
)

func main() {
    init, err := otelinit.ConfigureOpentelemetry()
    defer init.Shutdown()
}
```

Alternatively, to set the Honeycomb API key directly in code:

```go
import (
    "go.opentelemetry.io/contrib/otelinit"
    "github.com/honeycombio/otel-go/otelinit/honeycomb"
)

func main() {
    init, err := otelinit.ConfigureOpentelemetry(
        honeycomb.WithApiKey("api-key"),
    )
    defer init.Shutdown()
}
```

## Trade-offs and mitigations

This Initialization Layer is intentionally providing default configuration options, which may not precisely map out to the final configuration an end user desires. As such, there may be dependencies in the package that are not being used. For example, the default configuration will bring in both gRPC and HTTP exporters; using gRPC will result in HTTP being included but not used; similarly, using HTTP/protobuf will result in the gRPC dependency being pulled in but unused.

There still exists the ability to change the default configuration with environment variables or in-code configuration options, but there is not an easy way to remove unused dependencies without also keeping the minimal configuration option.

If removing unused dependencies is required, the user can avoid using the Initialization Layer and initialize OTel manually, which is the current experience today.

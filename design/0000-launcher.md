# Launcher

## What is it?

The Go Launcher (name TBD) is a configuration layer that chooses default values for configuration options that many OpenTelemetry users would ultimately configure manually, allowing for minimal code to quickly instrument with OpenTelemetry.

## Motivation

The goal of this Launcher is to help users that aren't familiar with OpenTelemetry quickly ramp up on what they need to get going and instrument their applications. [There is currently a lot of boilerplate code required to use OpenTelemetry](https://opentelemetry.io/docs/instrumentation/go/manual/#initializing-a-new-tracer), and users must make decisions early on before instrumentation works - for example, they must determine and find the appropriate exporters, set resource attributes, etc. This launcher is intended to simplify the setup and minimize decision-making overhead to be able to quickly get started.

Separately, vendors may choose to provide a vendor-specific package compatible with this launcher for easier exporting to their own backend. Neither the launcher nor the vendor-specific package are required to use OpenTelemetry, and using the launcher does not require the end user to use a vendor-specific package.

## Explanation

The Go Launcher provides a single function to reduce the amount of code that needs to be written to get started, and provides additional, straightforward options to add in that still make additional configuration easy.

The vendor-neutral launcher will live in the opentelemetry-go-contrib repo, and vendors may create separately hosted vendor-specific configuration packages that can be used with the launcher.

### Getting started

```shell
go get go.opentelemetry.io/contrib/launcher
```

### Configure

Minimal setup - by default will send all telemetry to localhost:4317, with the ability to set environment variables for endpoint, headers, resource attributes, and more as listed in the configuration options noted below:

```go
import "go.opentelemetry.io/contrib/launcher"

func main() {
    lnchr, err := launcher.ConfigureOpentelemetry()
    defer lnchr.Shutdown()
}
```

Or set headers directly in code instead:

```go
import "go.opentelemetry.io/contrib/launcher"

func main() {
    lnchr, err := launcher.ConfigureOpentelemetry(
        launcher.WithServiceName("service-name"),
        launcher.WithHeaders(map[string]string{
            "service-auth-key": "value",
            "service-useful-field": "testing",
        }),
    )
    defer lnchr.Shutdown()
}
```

### Configuration Options

| Config Option               | Env Variable                        | Required | Default              |
| --------------------------  | ----------------------------------- | -------- | -------------------- |
| WithServiceName             | OTEL_SERVICE_NAME                   | n*       | unknown_service:go   |
| WithServiceVersion          | OTEL_SERVICE_VERSION                | n        | -                    |
| WithHeaders                 | OTEL_EXPORTER_OTLP_HEADERS          | n        | {}                   |
| WithTracesExporterEndpoint  | OTEL_EXPORTER_OTLP_TRACES_ENDPOINT  | n        | localhost:4317       |
| WithTracesExporterInsecure  | OTEL_EXPORTER_OTLP_TRACES_INSECURE  | n        | false                |
| WithMetricsExporterEndpoint | OTEL_EXPORTER_OTLP_METRICS_ENDPOINT | n        | localhost:4317       |
| WithMetricsExporterInsecure | OTEL_EXPORTER_OTLP_METRICS_INSECURE | n        | false                |
| WithLogLevel                | OTEL_LOG_LEVEL                      | n        | info                 |
| WithPropagators             | OTEL_PROPAGATORS                    | n        | tracecontext,baggage |
| WithResourceAttributes      | OTEL_RESOURCE_ATTRIBUTES            | n        | -                    |
| WithMetricsReportingPeriod  | OTEL_EXPORTER_OTLP_METRICS_PERIOD   | n        | 30s                  |
| WithMetricsEnabled          | OTEL_METRICS_ENABLED                | n        | true                 |
| WithTracesEnabled           | OTEL_TRACES_ENABLED                 | n        | true                 |
| WithProtocol                | OTEL_EXPORTER_OTLP_PROTOCOL         | n        | grpc                 |

*Service name should be set using the `WithServiceName` configuration option, the `OTEL_SERVICE_NAME` environment variable, or by setting the `service.name` in `OTEL_RESOURCE_ATTRIBUTES`. The default service name is based on the SDK's behavior as it conforms to the specification: `unknown_service`, suffixed with either the process name (where possible) or `go`.

### Additional Configuration Options

Not yet implemented but still desired configuration options would include:

- `OTEL_EXPORTER_OTLP_PROTOCOL`: `http/protobuf`
    - ideally, selecting this would change the default endpoint to localhost:4318

### Using a Vendor-Specific Package

Vendors may create and maintain convenience configuration packages to more easily setup for export to a specific backend. For example, using a Honeycomb package would look like this:

Minimal setup, which sends to the Honeycomb endpoint and requires `HONEYCOMB_API_KEY` environment variable:

```go
import (
    "go.opentelemetry.io/contrib/launcher"
    _ "github.com/honeycombio/otel-launcher-go/launcher/honeycomb"
)

func main() {
    lnchr, err := launcher.ConfigureOpentelemetry()
    defer lnchr.Shutdown()
}
```

Alternatively, to set the Honeycomb API key directly in code:

```go
import (
    "go.opentelemetry.io/contrib/launcher"
    "github.com/honeycombio/otel-launcher-go/launcher/honeycomb"
)

func main() {
    lnchr, err := launcher.ConfigureOpentelemetry(
        honeycomb.WithApiKey("api-key"),
    )
    defer lnchr.Shutdown()
}
```

## Trade-offs and mitigations

This launcher is intentionally providing default configuration options, which may not precisely map out to the final configuration an end user desires. As such, there may be dependencies in the package that are not being used. For example, the default configuration will bring in both gRPC and HTTP exporters; using gRPC will result in HTTP being included but not used; similarly, using HTTP/protobuf will result in the gRPC dependency being pulled in but unused.

There still exists the ability to change the default configuration with environment variables or in-code configuration options, but there is not an easy way to remove unused dependencies without also keeping the minimal configuration option.

If removing unused dependencies is required, the user can avoid using the launcher and initialize OTel manually, which is the current experience today.

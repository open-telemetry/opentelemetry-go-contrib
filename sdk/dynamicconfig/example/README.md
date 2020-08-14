# Dynamic Metric Configuration
This example showcases the ability to configure metric collection periods at
runtime, via a remote configuration service. It is a prototype implementation of
[this experimental
specification](https://github.com/open-telemetry/opentelemetry-specification/blob/master/experimental/metrics/config-service.md). The prototype is currently available
only for Go.

**NOTE: this system is experimental**

## Push Controller
A prototype push controller that implements dynamic per-metric configuration is
available from the [Go contrib
repository](https://github.com/open-telemetry/opentelemetry-go-contrib). To
use it, ensure you have the following import

```go
import "go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/push"
```

Its use is almost identical to the push controller available from the Go SDK.
The crucial difference is that the constructor takes an additional argument
that represents the address of the configuration service. For example:

```go
pusher := push.New(
	simple.NewWithExactDistribution(),   // AggregatorSelector
	exporter,                            // Exporter
	"localhost:55700",                   // DIFFERENT: address of config service
	push.WithResource(resource),         // zero or more options
)
```

Otherwise, the contrib pusher is a drop-in replacement for the SDK's push
controller.

A simple example application is provided in [main.go](main.go).

## Collector Extension
A service backend for this configuration system has been implemented as an
extension on the [contrib
collector](https://github.com/open-telemetry/opentelemetry-collector-contrib).

To use it, include the following block under `extensions` in the collector's
startup configurations:

```yaml
dynamicconfig:
    endpoint: 0.0.0.0:55700               # listen on localhost:55700
    local_config_file: 'schedules.yaml'   # use metric config data from this file
    wait_time: 30                         # suggested time (in seconds) for client to wait between polls
```

Alternatively, one can specify a `remote_config_address` in place of the
`local_config_file`, if the collector would like to pull its metric data from
an upstream source (e.g. a third-party service, or another collector).


A sample configuration file is provided in [config/file-backend-config.yaml](config/file-backend-config.yaml),
and may be used directly with the contrib collector.

Additionally, we provide a sample [schedules.yaml](config/schedules.yaml)
that corresponds to metrics in the example app, and can be used directly with
the configuration file above. Details about setting up and modifying
this file can be found [here](https://github.com/open-telemetry/opentelemetry-specification/blob/master/experimental/metrics/config-service.md#local-file)

## Additional Resources
For a more detailed explanation of the concepts underlying this example,
please see the:

* [experimental specification](https://github.com/open-telemetry/opentelemetry-specification/blob/master/experimental/metrics/config-service.md)
* [experimental protocol](https://github.com/open-telemetry/opentelemetry-proto/pull/183)

# Feature Gates

The runtime package contains feature gates used to ease the migration
from the [previous runtime metrics conventions] to the new [OpenTelemetry Go
Runtime conventions], and to enable the opt-in metrics of those conventions.

Note that the new runtime metrics conventions are still experimental, and may
change in backwards incompatible ways as feedback is applied.

## Features

- [Include Deprecated Metrics](#include-deprecated-metrics)
- [Include Opt-In Metrics](#include-opt-in-metrics)

### Include Deprecated Metrics

To temporarily re-enable the deprecated metrics:

```console
export OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true
```

Eventually, the deprecated runtime metrics will be removed,
and setting the environment variable will no longer have any effect.

The value set must be the case-insensitive string of `"true"` to enable the
feature, and `"false"` to disable the feature. All other values are ignored.

### Include Opt-In Metrics

To produce the [opt-in metrics] of the Go runtime conventions, list them
comma-separated:

```console
export OTEL_GO_X_RUNTIME_METRICS_OPTIN=go.memory.gc.cycles,go.memory.gc.pause.duration,go.cpu.time
```

The supported metrics are:

- `go.memory.gc.cycles`, produced by `Start`.
- `go.cpu.time`, produced by `Start`.
- `go.memory.gc.pause.duration`, produced by `NewProducer`.

Unknown names are ignored. The same metrics can be enabled in code with the
`WithOptInMetrics` option; a metric is produced when either enables it. Once the metrics API supports the opt-in metric
advisory parameter, this environment variable will be replaced by it.

[opt-in metrics]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/metric-requirement-level.md#opt-in
[previous runtime metrics conventions]: https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/runtime@v0.52.0
[OpenTelemetry Go Runtime conventions]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/runtime/go-metrics.md

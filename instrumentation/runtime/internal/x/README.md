# Feature Gates

The runtime package contains feature gates used to enable the opt-in metrics of
the [OpenTelemetry Go Runtime conventions].

Note that the new runtime metrics conventions are still experimental, and may
change in backwards incompatible ways as feedback is applied.

## Features

- [Include Opt-In Metrics](#include-opt-in-metrics)

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

Unknown names are ignored. Once the metrics API supports the opt-in metric
advisory parameter, this environment variable will be replaced by it.

[opt-in metrics]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/metric-requirement-level.md#opt-in
[OpenTelemetry Go Runtime conventions]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/runtime/go-metrics.md

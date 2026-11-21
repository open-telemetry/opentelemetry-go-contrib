# Feature Gates

The otelhttp package contains feature gates used to enable opt-in metrics and
attributes from the [HTTP metrics semantic conventions].

These metrics are still experimental in the semantic conventions, and may
change in backwards incompatible ways as feedback is applied.

## Features

- [Include Opt-In Metrics](#include-opt-in-metrics)

### Include Opt-In Metrics

To produce opt-in metrics and attributes from the HTTP metrics conventions, list their names
comma-separated:

```console
export OTEL_GO_X_HTTP_METRICS_OPTIN=http.client.active_requests,url.scheme
```

The supported metrics and attributes are:

- `http.client.active_requests`, produced by `Transport`. A request is counted
  from the start of `RoundTrip` until the underlying `RoundTripper` returns.
- `url.scheme`, produced by `Transport` and included in client metric attributes.

Unknown names are ignored. Once the metrics API supports the opt-in metric
advisory parameter, this environment variable will be replaced by it.

[opt-in metrics]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/metric-requirement-level.md#opt-in
[HTTP metrics semantic conventions]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/http/http-metrics.md

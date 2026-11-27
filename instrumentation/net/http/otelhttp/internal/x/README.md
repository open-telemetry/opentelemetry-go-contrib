# Feature Gates

The otelhttp package contains feature gates used to enable the opt-in metrics
of the [HTTP metrics semantic conventions].

These metrics are still experimental in the semantic conventions, and may
change in backwards incompatible ways as feedback is applied.

## Features

- [Include Opt-In Metrics](#include-opt-in-metrics)
- [Known HTTP Methods](#known-http-methods)

### Include Opt-In Metrics

To produce the [opt-in metrics] of the HTTP metrics conventions, list them
comma-separated:

```console
export OTEL_GO_X_HTTP_METRICS_OPTIN=http.client.active_requests
```

The supported metrics are:

- `http.client.active_requests`, produced by `Transport`. A request is counted
  from the start of `RoundTrip` until the underlying `RoundTripper` returns.

Unknown names are ignored. Once the metrics API supports the opt-in metric
advisory parameter, this environment variable will be replaced by it.

[opt-in metrics]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/metric-requirement-level.md#opt-in
[HTTP metrics semantic conventions]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/http/http-metrics.md

### Known HTTP Methods

HTTP instrumentation recognizes a default set of HTTP request methods.
You can override this list using the
`OTEL_INSTRUMENTATION_HTTP_KNOWN_METHODS` environment variable.

Specify a comma-separated, case-sensitive list:

```console
export OTEL_INSTRUMENTATION_HTTP_KNOWN_METHODS=GET,POST,QUERY
```

When the environment variable is unset, the default known methods are:

- `CONNECT`
- `DELETE`
- `GET`
- `HEAD`
- `OPTIONS`
- `PATCH`
- `POST`
- `PUT`
- `TRACE`
- `QUERY`

When configured, the supplied list completely replaces the default list.
An empty value disables recognition of all methods.

Methods not present in the configured list are recorded as `_OTHER`.
For non-empty methods, the original method is preserved in
`http.request.method_original`.

The environment variable is read during package initialization and must
be configured before the application starts.

# Feature Gates

The `otelhttp` package contains feature gates used to enable opt-in metrics and attributes from the [HTTP semantic conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/http/http-metrics.md).

These metrics and attributes are opt-in in the semantic conventions and may change in backwards-incompatible ways as feedback is applied.

## Features

- [Include Opt-In Metrics](#include-opt-in-metrics)
- [Known HTTP Methods](#known-http-methods)

### Include Opt-In Metrics

To produce opt-in metrics and attributes from the HTTP conventions, list their names as a comma-separated value in `OTEL_GO_X_HTTP_METRICS_OPTIN`:

```console
export OTEL_GO_X_HTTP_METRICS_OPTIN=http.client.active_requests,http.client.request.body.size,http.server.request.body.size,http.server.response.body.size,url.scheme
```

The supported metrics are:

- `http.client.active_requests`, produced by `Transport`. A request is counted from the start of `RoundTrip` until the underlying `RoundTripper` returns.
- `http.client.request.body.size`, produced by `Transport`.
- `http.server.request.body.size` and `http.server.response.body.size`, produced by `Handler` and `Middleware`.
- `url.scheme`, produced by `Transport`.

Unknown names are ignored. Once the metrics API supports the opt-in metric advisory parameter, this environment variable will be replaced by it.

### Known HTTP Methods

The `OTEL_INSTRUMENTATION_HTTP_KNOWN_METHODS` environment variable configures which HTTP methods are recognized by the instrumentation.

By default, the following methods are recognized:

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

To override the default list, set the environment variable to a comma-separated list of methods:

```console
export OTEL_INSTRUMENTATION_HTTP_KNOWN_METHODS=GET,POST,PUT
```

This is a full override of the default list, not an additive configuration. Methods not included in the configured list are treated as unknown. Matching is case-sensitive.

The environment variable is read during package initialization. Set it before the application starts and before the package is initialized. Changing it after package initialization has no effect.

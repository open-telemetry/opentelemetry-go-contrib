# Feature Gates

The otelhttp package contains feature gates used to enable opt-in metrics and attributes of the [HTTP semantic conventions].

These metrics and attributes are opt-in in the semantic conventions and may change in backwards-incompatible ways as feedback is applied.

## Features

- [Include Opt-In Metrics](#include-opt-in-metrics)
- [Known HTTP Methods](#known-http-methods)

### Include Opt-In Metrics

To produce opt-in metrics and attributes of the HTTP conventions, list their names comma-separated:

```console
export OTEL_GO_X_HTTP_METRICS_OPTIN=http.client.active_requests,http.client.request.body.size,http.server.request.body.size,http.server.response.body.size,url.scheme.

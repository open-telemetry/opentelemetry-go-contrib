# HTTP Instrumentation

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)

This package provides OpenTelemetry tracing and metrics for HTTP servers and clients.

## Routes with request-copying middleware

`NewHandler` reads `http.Request.Pattern` after dispatch to identify the matched
route. Middleware that calls `WithContext` or `Clone` before a `http.ServeMux`
prevents the outer instrumentation from seeing the pattern set on that copy.

Enable `WithRouteTracking` on the enclosing `NewHandler` or `NewMiddleware`, and
place `RouteMiddleware` directly around the mux, inside the copying middleware:

```go
handler := otelhttp.NewHandler(
    withRequestID(otelhttp.RouteMiddleware(mux)),
    "server",
    otelhttp.WithRouteTracking(),
)
```

`RouteMiddleware` records the matched route using the enclosing instrumentation's
span name formatter and semantic conventions. Only the metric route is taken from
the routed request; other metric attributes and `WithMetricAttributesFn` retain
the request that the instrumentation passed to the next handler.
Nested route middlewares preserve the innermost matched pattern. Method and host
prefixes are removed from the `http.route` attribute. Unmatched requests do not
gain a route attribute or a span name derived from their URL path.

Deferred route capture preserves the span name and route attribute when a handler
panics. It does not recover the panic or change the instrumentation's response and
metric recording behavior during panic unwinding. Route tracking is disabled by
default. `RouteMiddleware` passes requests through unless the nearest enclosing
instrumented handler enables it. `WithRouteTracking` alone cannot observe a pattern
set on a copied request and has no effect on client transports.

See the [Go reference](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)
for a complete example and the server and client options.

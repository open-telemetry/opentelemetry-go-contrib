# Migrating from otelecho

The recommended replacement for this module is
[`github.com/labstack/echo-opentelemetry`](https://github.com/labstack/echo-opentelemetry).
It is not a drop-in replacement.

## Known incompatibilities

- `otelecho` instruments `github.com/labstack/echo/v4`. The replacement
  instruments `github.com/labstack/echo/v5`, so applications need to migrate
  Echo before switching middleware.
- `otelecho.Middleware(serverName, opts...)` is replaced by
  `echootel.NewMiddleware(serverName)` or
  `echootel.NewMiddlewareWithConfig(echootel.Config{...})`.
- Configuration moved from option functions to fields on `echootel.Config`.
  For example, `WithTracerProvider`, `WithMeterProvider`, `WithPropagators`,
  and `WithSkipper` map to fields with the same names.
- `WithMetricAttributeFn` and `WithEchoMetricAttributeFn` are replaced by
  `Config.MetricAttributes`, which receives the Echo context and extracted
  telemetry values.
- `WithOnError` is replaced by `Config.OnNextError`. The replacement also
  exposes `Config.OnExtractionError` for request extraction failures.

## Example

```go
import echootel "github.com/labstack/echo-opentelemetry"

e.Use(echootel.NewMiddlewareWithConfig(echootel.Config{
	ServerName:     "example.com",
	TracerProvider: tracerProvider,
	MeterProvider:  meterProvider,
	Propagators:    propagators,
	Skipper:        skipper,
}))
```

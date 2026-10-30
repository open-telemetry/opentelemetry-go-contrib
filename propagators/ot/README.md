# OpenTracing Propagator (deprecated)

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/propagators/ot.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/propagators/ot)

This module implements the `ot-tracer-*` and `ot-baggage-*` propagation formats used by the default OpenTracing tracer.

## Deprecation

This module and package are deprecated and will be removed in v1.49.0.
OpenTelemetry has [deprecated OpenTracing compatibility requirements](https://opentelemetry.io/blog/2026/deprecating-opentracing-compatibility/).
Removal is tracked in [#10046](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/10046).

## Migration

[Migrate from OpenTracing](https://opentelemetry.io/docs/compatibility/migration/opentracing/) to native OpenTelemetry APIs and SDKs.
Use [`propagation.TraceContext`](https://pkg.go.dev/go.opentelemetry.io/otel/propagation#TraceContext) and [`propagation.Baggage`](https://pkg.go.dev/go.opentelemetry.io/otel/propagation#Baggage) for W3C Trace Context and W3C Baggage propagation.
When using `autoprop`, configure `OTEL_PROPAGATORS=tracecontext,baggage` instead of `ottrace`.

Coordinate propagation changes across communicating services: the W3C formats use different headers and do not read the legacy OpenTracing headers.
The existing propagator and the `autoprop` registration for `ottrace` remain available during deprecation to support migration.

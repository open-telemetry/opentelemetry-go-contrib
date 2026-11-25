# Prometheus exporter example

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/examples/prometheus.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/examples/prometheus)

Exposes OpenTelemetry metrics through a Prometheus scrape endpoint.

From the repository root:

```sh
cd examples/prometheus
go run .
```

In another terminal, run `curl http://localhost:2223/metrics`. Stop the program with Ctrl+C.

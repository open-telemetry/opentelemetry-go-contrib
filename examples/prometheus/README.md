# Prometheus exporter example

Exposes OpenTelemetry metrics through a Prometheus scrape endpoint.

From the repository root:

```sh
cd examples/prometheus
go run .
```

In another terminal, run `curl http://localhost:2223/metrics`. Stop the program with Ctrl+C.

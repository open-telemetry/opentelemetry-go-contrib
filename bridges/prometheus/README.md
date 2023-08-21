# Prometheus Bridge

Status: Experimental

The Prometheus Bridge allows using the Prometheus Golang client library
(github.com/prometheus/client_golang) with the OpenTelemetry SDK.

## Usage

```golang
// Make a Promethes bridge "Metric Producer" that adds metrics from the
// Prometheus DefaultGatherer. Add the WithGatherer(registry) option to add
// metrics from other registries.
bridge := prombridge.NewMetricProducer()
// Make a Periodic Reader to periodically gather metrics from the bridge, and
// push to an OpenTelemetry exporter.
reader := metric.NewPeriodicReader(otelExporter, metric.WithProducer(bridge))
// Create an OTel MeterProvider with our reader. Metrics from OpenTelemetry
// instruments are combined with metrics from Prometheus instruments in
// exported batches of metrics.
mp := metric.NewMeterProvider(metric.WithReader(reader))
```

## Limitations

* Summary metrics are dropped by the bridge.
* Start times for histograms and counters are set to the process start time.
* It does not currently support exponential histograms.

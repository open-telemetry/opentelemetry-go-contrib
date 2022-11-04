# Jaeger Remote Sampler

This package implements [Jaeger remote sampler](https://www.jaegertracing.io/docs/latest/sampling/#collector-sampling-configuration).

## Usage

Configuration in the code:

```go
	jaegerRemoteSampler := jaegerremote.New(
		"your-service-name",
		jaegerremote.WithSamplingServerURL("http://{sampling_service_host_name}:5778"),
		jaegerremote.WithSamplingRefreshInterval(10*time.Second),
		jaegerremote.WithInitialSampler(trace.TraceIDRatioBased(0.5)),
	)

	tp := trace.NewTracerProvider(
		trace.WithSampler(jaegerRemoteSampler),
		...
	)
	otel.SetTracerProvider(tp)
```

Notes:

* At this time, the Jaeger Remote Sampler can only be configured in the code,
  configuration via `OTEL_TRACES_SAMPLER=jaeger_sampler` environment variable is not supported.
* Service name must be passed to the constructor. It will be used by the sampler to poll
  the backend for the sampling strategy for this service.
* Both Jaeger Agent and OpenTelemetry Collector implement the Jaeger sampling service endpoint.

## Example

[example/](./example) shows how to host remote sampling strategies using the OpenTelemetry Collector.
The Collector uses the Jaeger receiver to host the strategy file. Note you do not need to run Jaeger to make use of the Jaeger remote sampling protocol. However, you do need Jaeger backend if you want to utilize its adaptive sampling engine that auto-calculates remote sampling strategies.

Run the OpenTelemetry Collector using docker-compose:

```shell
$ docker-compose up -d
```

You can fetch the strategy file using curl:

```shell
$ curl 'localhost:5778/sampling?service=foo'
$ curl 'localhost:5778/sampling?service=myService'
```

Run the Go program.
This program will start with an initial sampling percentage of 50% and tries to fetch the sampling strategies from the OpenTelemetry Collector.
It will print the entire Jaeger remote sampler structure every 10 seconds, this allows you to observe the internal sampler.

```shell
$ go run .
```

## Update generated Jaeger code

Code is generated using the .proto files from [jaeger-idl](https://github.com/jaegertracing/jaeger-idl).
In case [sampling.proto](./jaeger-idl/proto/api_v2/sampling.proto) is modified these have to be regenerated.

```shell
$ make proto-gen
```

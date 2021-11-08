# Jaeger Remote Sampler

This package implements [Jaeger remote sampler](https://www.jaegertracing.io/docs/latest/sampling/#collector-sampling-configuration).

## Example

[example/](./example) shows how to host remote sampling strategies using the OpenTelemetry Collector.
The Collector uses the Jaeger receiver to host the strategy file, note you do not need to send Jaeger to make use of the Jaeger remote sampling protocol. 

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

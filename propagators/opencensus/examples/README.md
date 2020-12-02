# OpenCensus binary propagation example

The server uses OpenTelemetry with the OpenCensus binary propagation format.
The client uses OpenCensus, which is hard-coded to use the OpenCensus binary
propagation format. Since the client and server use the same propagation
format, the ParentSpanID from the server spans should match the SpanID from
the client spans, and both should share the same TraceID.

### Usage

First, start the opentelemetry server:
```bash
go run opentelemetry_server/server.go
```

In another shell, start the OpenCensus client:
```bash
go run opencensus_client/client.go
```

### Example Client Output

```
Configuring OpenCensus, and registering the Print exporter.

TraceID:      9d59b1bdbde34cdaac6cfb5b8f3c4685
SpanID:       07733a2559ef492d
...
Greeting: Hello world
```

Note that there is no ParentSpanID listed in the client.

### Example Server Output

```
Registering opentelemetry stdout exporter.
Starting the GRPC server, and using the OpenCensus binary propagation format.
[
	{
		"SpanContext": {
			"TraceID": "9d59b1bdbde34cdaac6cfb5b8f3c4685",
			"SpanID": "94738571415fdb63",
			"TraceFlags": 1
		},
		"ParentSpanID": "07733a2559ef492d",
        ...
    }
]
```

The TraceID matches the TraceID from the OpenCensus client span, and the ParentSpanID matches the SpanID of the OpenCensus client span.

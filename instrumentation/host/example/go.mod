module go.opentelemetry.io/contrib/instrumentation/host/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../
	go.opentelemetry.io/contrib/instrumentation/host => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/host v0.23.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.23.1-0.20210928160814-00d8ca5890a8
	go.opentelemetry.io/otel/metric v0.23.1-0.20210928160814-00d8ca5890a8
	go.opentelemetry.io/otel/sdk/export/metric v0.23.1-0.20210928160814-00d8ca5890a8 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.23.1-0.20210928160814-00d8ca5890a8
)

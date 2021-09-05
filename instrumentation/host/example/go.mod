module go.opentelemetry.io/contrib/instrumentation/host/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../
	go.opentelemetry.io/contrib/instrumentation/host => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/host v0.22.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.23.0
	go.opentelemetry.io/otel/metric v0.23.0
	go.opentelemetry.io/otel/sdk/metric v0.23.0
)

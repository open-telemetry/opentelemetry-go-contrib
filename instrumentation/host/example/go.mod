module go.opentelemetry.io/contrib/instrumentation/host/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../
	go.opentelemetry.io/contrib/instrumentation/host => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/host v0.31.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.29.0
	go.opentelemetry.io/otel/metric v0.29.0
	go.opentelemetry.io/otel/sdk/metric v0.29.1-0.20220425151224-b8e4241a32f2
)

module go.opentelemetry.io/contrib/instrumentation/runtime/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../
	go.opentelemetry.io/contrib/instrumentation/runtime => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/runtime v0.31.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.30.0
	go.opentelemetry.io/otel/metric v0.30.0
	go.opentelemetry.io/otel/sdk/metric v0.30.0
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
)

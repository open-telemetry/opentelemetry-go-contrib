module go.opentelemetry.io/contrib/instrumentation/runtime/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/runtime => ../

require (
	go.opentelemetry.io/contrib/instrumentation/runtime v0.43.0
	go.opentelemetry.io/otel v1.17.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.40.0
	go.opentelemetry.io/otel/sdk v1.17.0
	go.opentelemetry.io/otel/sdk/metric v0.40.0
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.17.0 // indirect
	go.opentelemetry.io/otel/trace v1.17.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
)

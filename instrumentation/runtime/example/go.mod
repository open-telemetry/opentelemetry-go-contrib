module go.opentelemetry.io/contrib/instrumentation/runtime/example

go 1.21

replace go.opentelemetry.io/contrib/instrumentation/runtime => ../

require (
	go.opentelemetry.io/contrib/instrumentation/runtime v0.49.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/sdk/metric v1.24.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
)

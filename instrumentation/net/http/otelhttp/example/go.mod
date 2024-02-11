module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example

go 1.20

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.48.0
	go.opentelemetry.io/otel v1.23.1
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.23.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.23.0
	go.opentelemetry.io/otel/sdk v1.23.1
	go.opentelemetry.io/otel/sdk/metric v1.23.1
	go.opentelemetry.io/otel/trace v1.23.1
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.23.1 // indirect
	golang.org/x/sys v0.16.0 // indirect
)

module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.41.0
	go.opentelemetry.io/otel v1.15.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.38.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.15.0
	go.opentelemetry.io/otel/metric v0.38.0
	go.opentelemetry.io/otel/sdk v1.15.0
	go.opentelemetry.io/otel/sdk/metric v0.38.0
	go.opentelemetry.io/otel/trace v1.15.0
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

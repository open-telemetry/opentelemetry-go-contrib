module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example

go 1.23.0

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0
	go.opentelemetry.io/otel v1.37.1-0.20250827112407-3342341f1508
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.37.1-0.20250827112407-3342341f1508
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.37.1-0.20250827112407-3342341f1508
	go.opentelemetry.io/otel/sdk v1.37.1-0.20250827112407-3342341f1508
	go.opentelemetry.io/otel/sdk/metric v1.37.1-0.20250827112407-3342341f1508
	go.opentelemetry.io/otel/trace v1.37.1-0.20250827112407-3342341f1508
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.1-0.20250827112407-3342341f1508 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

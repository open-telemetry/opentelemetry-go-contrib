module go.opentelemetry.io/contrib/examples/dice/instrumented

go 1.22.0

require (
	go.opentelemetry.io/contrib/bridges/otelslog v0.9.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.11.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.35.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.35.0
	go.opentelemetry.io/otel/log v0.11.0
	go.opentelemetry.io/otel/metric v1.35.0
	go.opentelemetry.io/otel/sdk v1.35.0
	go.opentelemetry.io/otel/sdk/log v0.11.0
	go.opentelemetry.io/otel/sdk/metric v1.35.0
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

replace (
	go.opentelemetry.io/contrib/bridges/otelslog => ../../../bridges/otelslog
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../instrumentation/net/http/otelhttp
)

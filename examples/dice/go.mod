module go.opentelemetry.io/contrib/examples/dice

go 1.22

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.55.0
	go.opentelemetry.io/otel v1.30.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.6.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.30.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.30.0
	go.opentelemetry.io/otel/log v0.6.1-0.20240921150428-063239fa3703
	go.opentelemetry.io/otel/metric v1.30.0
	go.opentelemetry.io/otel/sdk v1.30.0
	go.opentelemetry.io/otel/sdk/log v0.6.1-0.20240921150428-063239fa3703
	go.opentelemetry.io/otel/sdk/metric v1.30.0
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel/trace v1.30.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../instrumentation/net/http/otelhttp

module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example

go 1.21

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.51.0
	go.opentelemetry.io/otel v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/sdk v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/sdk/metric v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/trace v1.26.1-0.20240520052501-49c866fbcd20
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.26.1-0.20240520052501-49c866fbcd20 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

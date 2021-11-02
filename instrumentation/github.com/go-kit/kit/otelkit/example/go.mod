module go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/example

go 1.15

require (
	github.com/gorilla/mux v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit v0.26.0
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.1.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/trace v1.1.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit => ../
)

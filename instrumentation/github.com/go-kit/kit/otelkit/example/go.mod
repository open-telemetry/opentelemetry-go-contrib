module go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/example

go 1.14

require (
	github.com/gorilla/mux v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit v0.19.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit => ../
	go.opentelemetry.io/contrib/propagators => ../../../../../../propagators
)

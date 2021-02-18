module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.17.0
	go.opentelemetry.io/otel v0.17.0
	go.opentelemetry.io/otel/exporters/stdout v0.17.0
	go.opentelemetry.io/otel/sdk v0.17.0
	go.opentelemetry.io/otel/trace v0.17.0
)

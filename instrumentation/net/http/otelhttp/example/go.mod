module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.29.0
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.5.0
	go.opentelemetry.io/otel/sdk v1.5.0
	go.opentelemetry.io/otel/trace v1.5.0
)

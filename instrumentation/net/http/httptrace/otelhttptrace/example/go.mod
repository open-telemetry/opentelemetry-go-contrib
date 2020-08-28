module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../otelhttp
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.11.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/stdout v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
)

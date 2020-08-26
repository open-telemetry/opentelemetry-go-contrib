module go.opentelemetry.io/contrib/instrumentation/net/http/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../
	go.opentelemetry.io/contrib/instrumentation/net/http => ../
)

require (
	go.opentelemetry.io/contrib/instrumentation/net/http v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/stdout v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
)

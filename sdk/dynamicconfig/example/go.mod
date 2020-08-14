module go.opentelemetry.io/contrib/sdk/dynamicconfig/example

go 1.13

replace go.opentelemetry.io/contrib => ../../../

replace go.opentelemetry.io/contrib/sdk/dynamicconfig => ../

require (
	go.opentelemetry.io/contrib/sdk/dynamicconfig v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/otlp v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
)

module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/example

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

replace go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama => ../

require (
	github.com/Shopify/sarama v1.27.0
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/stdout v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
)

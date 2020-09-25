module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama/example

go 1.14

replace go.opentelemetry.io/contrib => ../../../../../..

replace go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama => ../

require (
	github.com/Shopify/sarama v1.27.0
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama v0.12.0
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/stdout v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
)

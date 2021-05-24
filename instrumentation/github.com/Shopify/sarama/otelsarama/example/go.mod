module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama => ../
)

require (
	github.com/Shopify/sarama v1.29.0
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)

module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/example

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

replace go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama => ../

require (
	github.com/Shopify/sarama v1.26.4
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama v0.10.0
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/stdout v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
	google.golang.org/grpc v1.31.0
)

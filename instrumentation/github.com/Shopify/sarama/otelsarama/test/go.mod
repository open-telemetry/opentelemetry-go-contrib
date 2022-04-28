module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama/test

go 1.16

require (
	github.com/Shopify/sarama v1.32.0
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama v0.31.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/sdk v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama => ../

replace go.opentelemetry.io/contrib => ../../../../../../

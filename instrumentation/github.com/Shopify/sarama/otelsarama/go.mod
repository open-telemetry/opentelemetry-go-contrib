module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/Shopify/sarama v1.27.2
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.14.0
	go.opentelemetry.io/otel v0.14.0
)

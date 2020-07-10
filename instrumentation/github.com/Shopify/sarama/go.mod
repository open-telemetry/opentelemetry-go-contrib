module go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama

go 1.14

replace go.opentelemetry.io/contrib => ../../../..

require (
	github.com/Shopify/sarama v1.26.4
	github.com/google/uuid v1.1.1
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.7.0
	go.opentelemetry.io/otel v0.8.0
	google.golang.org/grpc v1.30.0
)

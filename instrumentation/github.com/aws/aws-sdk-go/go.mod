module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go

go 1.15

require (
	github.com/aws/aws-sdk-go v1.35.3
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.13.0
	go.opentelemetry.io/otel v0.13.0
)

replace (
	go.opentelemetry.io/contrib => ../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../net/http/otelhttp
	go.opentelemetry.io/contrib/propagators => ../../../../propagators
)

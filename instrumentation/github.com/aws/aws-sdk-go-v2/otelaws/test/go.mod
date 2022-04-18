module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/test

go 1.16

require (
	github.com/aws/aws-sdk-go-v2 v1.16.2
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.15.3
	github.com/aws/aws-sdk-go-v2/service/route53 v1.20.3
	github.com/aws/smithy-go v1.11.2
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.31.0
	go.opentelemetry.io/otel v1.6.3
	go.opentelemetry.io/otel/sdk v1.6.3
	go.opentelemetry.io/otel/trace v1.6.3
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

replace go.opentelemetry.io/contrib => ../../../../../../

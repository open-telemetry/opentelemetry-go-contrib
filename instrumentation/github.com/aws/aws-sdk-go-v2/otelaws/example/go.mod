module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../
)

require (
	github.com/aws/aws-sdk-go-v2 v1.9.2
	github.com/aws/aws-sdk-go-v2/config v1.8.3
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.5.2
	github.com/aws/aws-sdk-go-v2/service/s3 v1.16.1
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.25.0
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.1
	go.opentelemetry.io/otel/sdk v1.0.1
	go.opentelemetry.io/otel/trace v1.0.1
)

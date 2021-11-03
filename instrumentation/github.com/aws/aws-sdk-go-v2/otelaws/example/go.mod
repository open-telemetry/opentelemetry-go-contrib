module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../
)

require (
	github.com/aws/aws-sdk-go-v2 v1.10.0
	github.com/aws/aws-sdk-go-v2/config v1.9.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.6.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.17.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.26.0
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.1.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/trace v1.1.0
)

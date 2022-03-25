module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../
)

require (
	github.com/aws/aws-sdk-go-v2 v1.15.0
	github.com/aws/aws-sdk-go-v2/config v1.15.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.15.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.26.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.30.0
	go.opentelemetry.io/otel v1.6.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.6.0
	go.opentelemetry.io/otel/sdk v1.6.0
	go.opentelemetry.io/otel/trace v1.6.0
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
)

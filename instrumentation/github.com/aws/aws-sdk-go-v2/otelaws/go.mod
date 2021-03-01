module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators v0.17.0 => ../../../../../propagators
)

require (
	github.com/aws/aws-sdk-go-v2 v1.2.0
	github.com/aws/aws-sdk-go-v2/config v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.1.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.2.0 // indirect
	github.com/aws/smithy-go v1.1.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.17.0
	go.opentelemetry.io/contrib/propagators v0.17.0
	go.opentelemetry.io/contrib/propagators/aws v0.17.0 // indirect
	go.opentelemetry.io/otel v0.17.0
	go.opentelemetry.io/otel/exporters/otlp v0.17.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout v0.17.0 // indirect
	go.opentelemetry.io/otel/oteltest v0.17.0
	go.opentelemetry.io/otel/trace v0.17.0
	google.golang.org/grpc v1.36.0 // indirect
)

module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/test

go 1.19

require (
	github.com/aws/aws-sdk-go-v2 v1.18.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.19.5
	github.com/aws/aws-sdk-go-v2/service/route53 v1.27.7
	github.com/aws/smithy-go v1.13.5
	github.com/stretchr/testify v1.8.2
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.41.0
	go.opentelemetry.io/otel v1.15.0
	go.opentelemetry.io/otel/sdk v1.15.0
	go.opentelemetry.io/otel/trace v1.15.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.26 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.26 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.20.8 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

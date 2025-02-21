module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/test

go 1.22.0

require (
	github.com/aws/aws-sdk-go-v2 v1.36.2
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.40.2
	github.com/aws/aws-sdk-go-v2/service/route53 v1.48.8
	github.com/aws/smithy-go v1.22.3
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.59.0
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/sdk v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.33 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.33 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.33.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.37.15 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

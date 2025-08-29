module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.23.0

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

require (
	github.com/aws/aws-sdk-go-v2 v1.38.2
	github.com/aws/aws-sdk-go-v2/config v1.31.5
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.50.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.87.2
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.62.0
	go.opentelemetry.io/otel v1.37.1-0.20250828230916-d99c68cb21b2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.37.1-0.20250825143334-4b2bef6dd972
	go.opentelemetry.io/otel/sdk v1.37.1-0.20250825143334-4b2bef6dd972
	go.opentelemetry.io/otel/trace v1.37.1-0.20250825143334-4b2bef6dd972
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.9 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.8.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.38.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.34.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.1 // indirect
	github.com/aws/smithy-go v1.23.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.1-0.20250825143334-4b2bef6dd972 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

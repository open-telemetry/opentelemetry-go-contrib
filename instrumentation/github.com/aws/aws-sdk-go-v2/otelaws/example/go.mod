module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.17

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

require (
	github.com/aws/aws-sdk-go-v2 v1.16.14
	github.com/aws/aws-sdk-go-v2/config v1.17.5
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.16.4
	github.com/aws/aws-sdk-go-v2/service/s3 v1.27.9
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.34.0
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.9.0
	go.opentelemetry.io/otel/sdk v1.9.0
	go.opentelemetry.io/otel/trace v1.9.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.18 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.17 // indirect
	github.com/aws/smithy-go v1.13.2 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
)

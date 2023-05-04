module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

require (
	github.com/aws/aws-sdk-go-v2 v1.18.0
	github.com/aws/aws-sdk-go-v2/config v1.18.22
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.19.6
	github.com/aws/aws-sdk-go-v2/service/s3 v1.33.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.41.1
	go.opentelemetry.io/otel v1.16.0-rc.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.16.0-rc.1
	go.opentelemetry.io/otel/sdk v1.16.0-rc.1
	go.opentelemetry.io/otel/trace v1.16.0-rc.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.33 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.25 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.28 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.14.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.20.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.10 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0-rc.1 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

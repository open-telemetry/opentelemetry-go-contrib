module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

go 1.26.0

replace go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../

require (
	github.com/aws/aws-sdk-go-v2 v1.48.1
	github.com/aws/aws-sdk-go-v2/config v1.33.22
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.74.2
	github.com/aws/aws-sdk-go-v2/service/s3 v1.116.1
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.72.0
	go.opentelemetry.io/otel v1.47.1-0.20261127133057-2b79af723220
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.1-0.20261119094326-34915e1fc1ec
	go.opentelemetry.io/otel/sdk v1.47.1-0.20261119094326-34915e1fc1ec
	go.opentelemetry.io/otel/trace v1.47.1-0.20261119094326-34915e1fc1ec
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.28 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.21.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.26 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.48.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.53.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.39.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.44.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.53.1 // indirect
	github.com/aws/smithy-go v1.28.10 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.2 // indirect
	go.opentelemetry.io/otel/log v1.47.1-0.20261119094326-34915e1fc1ec // indirect
	go.opentelemetry.io/otel/metric v1.47.1-0.20261119094326-34915e1fc1ec // indirect
	golang.org/x/sys v0.50.0 // indirect
)

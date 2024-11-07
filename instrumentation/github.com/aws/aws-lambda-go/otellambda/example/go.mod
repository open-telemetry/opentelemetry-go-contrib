module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/example

go 1.22

replace (
	go.opentelemetry.io/contrib/detectors/aws/lambda => ../../../../../../detectors/aws/lambda
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda => ../
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../../../aws-sdk-go-v2/otelaws
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../../net/http/otelhttp
)

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2/config v1.28.2
	github.com/aws/aws-sdk-go-v2/service/s3 v1.66.3
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.56.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.56.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.56.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.56.0
	go.opentelemetry.io/otel v1.31.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.31.0
	go.opentelemetry.io/otel/sdk v1.31.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.32.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.6 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.43 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.36.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.36.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.32.4 // indirect
	github.com/aws/smithy-go v1.22.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
)

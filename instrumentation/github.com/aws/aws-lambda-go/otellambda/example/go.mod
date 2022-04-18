module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/example

go 1.16

replace (
	go.opentelemetry.io/contrib/detectors/aws/lambda => ../../../../../../detectors/aws/lambda
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda => ../
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws => ../../../aws-sdk-go-v2/otelaws
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../../net/http/otelhttp
)

require (
	github.com/aws/aws-lambda-go v1.30.0
	github.com/aws/aws-sdk-go-v2/config v1.15.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.26.4
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.31.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.31.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.31.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.31.0
	go.opentelemetry.io/otel v1.6.3
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.6.3
	go.opentelemetry.io/otel/sdk v1.6.3
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

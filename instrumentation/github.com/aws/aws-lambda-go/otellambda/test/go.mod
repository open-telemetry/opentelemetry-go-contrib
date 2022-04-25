module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/test

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/detectors/aws/lambda => ../../../../../../detectors/aws/lambda
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda => ../
	go.opentelemetry.io/contrib/propagators/aws => ../../../../../../propagators/aws
)

require (
	github.com/aws/aws-lambda-go v1.30.0
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.31.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.31.0
	go.opentelemetry.io/contrib/propagators/aws v1.6.0
	go.opentelemetry.io/otel v1.6.4-0.20220425151224-b8e4241a32f2
	go.opentelemetry.io/otel/sdk v1.6.3
	go.opentelemetry.io/otel/trace v1.6.3
)

module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/test

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/detectors/aws/lambda => ../../../../../../detectors/aws/lambda
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda => ../
	go.opentelemetry.io/contrib/propagators/aws => ../../../../../../propagators/aws
)

require (
	github.com/aws/aws-lambda-go v1.27.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.26.1
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.26.1
	go.opentelemetry.io/contrib/propagators/aws v1.1.1
	go.opentelemetry.io/otel v1.1.0
	// ToDo: update go.opentelemetry.io/otel/sdk package version
	go.opentelemetry.io/otel/sdk v1.1.1-0.20211105153457-6d2aeb0dc3dd
	go.opentelemetry.io/otel/trace v1.1.0
)

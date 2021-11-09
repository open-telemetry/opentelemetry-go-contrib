module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda

go 1.15

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/aws/aws-lambda-go v1.27.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.1.0
	// ToDo: update go.opentelemetry.io/otel/sdk package version
	go.opentelemetry.io/otel/sdk v1.1.1-0.20211105153457-6d2aeb0dc3dd
	go.opentelemetry.io/otel/trace v1.1.0
)

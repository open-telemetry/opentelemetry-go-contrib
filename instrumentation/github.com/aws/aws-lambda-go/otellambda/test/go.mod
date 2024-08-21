module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/test

go 1.21

replace (
	go.opentelemetry.io/contrib/detectors/aws/lambda => ../../../../../../detectors/aws/lambda
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda => ../
	go.opentelemetry.io/contrib/propagators/aws => ../../../../../../propagators/aws
)

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.53.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.53.0
	go.opentelemetry.io/contrib/propagators/aws v1.28.0
	go.opentelemetry.io/otel v1.28.1-0.20240821153356-fe6c67e7e9b4
	go.opentelemetry.io/otel/sdk v1.28.1-0.20240821153356-fe6c67e7e9b4
	go.opentelemetry.io/otel/trace v1.28.1-0.20240821153356-fe6c67e7e9b4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.1-0.20240821153356-fe6c67e7e9b4 // indirect
	golang.org/x/sys v0.24.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

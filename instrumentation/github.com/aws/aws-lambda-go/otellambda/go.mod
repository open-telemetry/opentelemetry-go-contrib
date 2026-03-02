module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda

go 1.24.0

replace (
	go.opentelemetry.io/contrib/detectors/aws/lambda => ../../../../../detectors/aws/lambda
	go.opentelemetry.io/contrib/propagators/aws => ../../../../../propagators/aws
)

require (
	github.com/aws/aws-lambda-go v1.52.0
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.65.0
	go.opentelemetry.io/contrib/propagators/aws v1.40.0
	go.opentelemetry.io/otel v1.40.0
	go.opentelemetry.io/otel/sdk v1.40.0
	go.opentelemetry.io/otel/trace v1.40.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

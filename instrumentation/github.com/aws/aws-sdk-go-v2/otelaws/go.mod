module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	github.com/aws/aws-sdk-go-v2 v1.2.0
	github.com/aws/aws-sdk-go-v2/service/route53 v1.1.1
	github.com/aws/smithy-go v1.1.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.18.0
	go.opentelemetry.io/otel v0.18.0
	go.opentelemetry.io/otel/oteltest v0.18.0
	go.opentelemetry.io/otel/trace v0.18.0
)

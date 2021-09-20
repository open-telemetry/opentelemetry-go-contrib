module go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/test

go 1.15

require (
	github.com/go-kit/kit v0.11.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit v0.24.0
	go.opentelemetry.io/otel v1.0.0
	go.opentelemetry.io/otel/sdk v1.0.0
	go.opentelemetry.io/otel/trace v1.0.0
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit => ../

replace go.opentelemetry.io/contrib => ../../../../../../

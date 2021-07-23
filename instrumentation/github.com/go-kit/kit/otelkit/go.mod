module go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit

go 1.15

require (
	github.com/go-kit/kit v0.11.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/oteltest v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC1
)

replace go.opentelemetry.io/contrib => ../../../../../

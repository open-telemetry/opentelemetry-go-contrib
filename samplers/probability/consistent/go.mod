module go.opentelemetry.io/contrib/samplers/probability/consistent

go 1.15

require (
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/trace v1.2.0
)

replace go.opentelemetry.io/otel => ../../../../go

replace go.opentelemetry.io/otel/trace => ../../../../go/trace

replace go.opentelemetry.io/otel/sdk => ../../../../go/sdk

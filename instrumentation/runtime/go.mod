module go.opentelemetry.io/contrib/instrumentation/runtime

go 1.16

replace go.opentelemetry.io/contrib => ../..

require (
	github.com/hashicorp/go-multierror v1.1.1
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/metric v0.30.0
	go.opentelemetry.io/otel/sdk/metric v0.30.0
)

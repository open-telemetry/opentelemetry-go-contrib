module go.opentelemetry.io/contrib/instrumentation/runtime

go 1.14

replace go.opentelemetry.io/contrib => ../..

require (
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/stdout v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
)

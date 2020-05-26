module go.opentelemetry.io/contrib/plugins/runtime

go 1.14

replace go.opentelemetry.io/otel => ../../../../../go.opentelemetry.io

replace go.opentelemetry.io/contrib => ../../

require (
	github.com/stretchr/testify v1.4.0
	go.opentelemetry.io/otel v0.6.0
)

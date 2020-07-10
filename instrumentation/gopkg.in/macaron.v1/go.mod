module go.opentelemetry.io/contrib/instrumentation/macaron

go 1.14

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.8.0
	go.opentelemetry.io/otel v0.8.0
	gopkg.in/macaron.v1 v1.3.9
)

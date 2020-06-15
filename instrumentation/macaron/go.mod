module go.opentelemetry.io/contrib/instrumentation/macaron

go 1.14

require (
	github.com/stretchr/testify v1.5.1
	go.opentelemetry.io/contrib v0.6.1
	go.opentelemetry.io/otel v0.6.0
	gopkg.in/macaron.v1 v1.3.9
)

replace go.opentelemetry.io/contrib => ../../

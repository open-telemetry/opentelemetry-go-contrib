module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../..
	go.opentelemetry.io/contrib/propagators => ../../../../propagators
)

require (
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.15.0
	go.opentelemetry.io/contrib/propagators v0.15.0
	go.opentelemetry.io/otel v0.15.0
	gopkg.in/macaron.v1 v1.3.9
)

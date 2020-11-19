module go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron => ../
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron v0.13.0
	go.opentelemetry.io/otel v0.13.0
	go.opentelemetry.io/otel/exporters/stdout v0.13.0
	go.opentelemetry.io/otel/sdk v0.13.0
	gopkg.in/macaron.v1 v1.4.0
)

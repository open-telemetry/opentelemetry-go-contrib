module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../..
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../propagators/b3
)

require (
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/propagators/b3 v1.6.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
	gopkg.in/macaron.v1 v1.4.0
)

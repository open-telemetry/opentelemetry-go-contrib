module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../..
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../propagators/b3
)

require (
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.22.0
	go.opentelemetry.io/contrib/propagators/b3 v0.22.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/trace v1.0.0-RC3
	gopkg.in/macaron.v1 v1.4.0
)

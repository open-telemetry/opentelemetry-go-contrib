module go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3
)

require (
	github.com/labstack/echo/v4 v4.7.2
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/propagators/b3 v1.6.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
)

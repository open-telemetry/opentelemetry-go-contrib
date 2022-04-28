module go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho/test

go 1.16

require (
	github.com/labstack/echo/v4 v4.7.2
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.31.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/sdk v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho => ../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

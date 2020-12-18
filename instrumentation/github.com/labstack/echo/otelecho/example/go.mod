module go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/labstack/echo/otelecho/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho => ../
	go.opentelemetry.io/contrib/propagators => ../../../../../../propagators
)

require (
	github.com/labstack/echo/v4 v4.1.17
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.15.1
	go.opentelemetry.io/otel v0.15.0
	go.opentelemetry.io/otel/exporters/stdout v0.15.0
	go.opentelemetry.io/otel/sdk v0.15.0
)

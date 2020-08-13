module go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo

go 1.14

replace go.opentelemetry.io/contrib => ../../../..

require (
	github.com/labstack/echo/v4 v4.1.16
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.10.1
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/stdout v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
	google.golang.org/grpc v1.31.0
)

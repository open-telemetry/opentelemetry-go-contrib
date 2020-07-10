module go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo

go 1.14

replace go.opentelemetry.io/contrib => ../../../..

require (
	github.com/labstack/echo/v4 v4.1.16
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.8.0
	go.opentelemetry.io/otel v0.8.0
	google.golang.org/grpc v1.30.0
)

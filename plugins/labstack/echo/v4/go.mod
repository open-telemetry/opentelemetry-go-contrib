module go.opentelemetry.io/contrib/plugins/labstack/echo/v4

go 1.14

replace go.opentelemetry.io/contrib => ../../../..

replace go.opentelemetry.io/contrib/plugins/labstack/echo/internal => ../internal

require (
	github.com/labstack/echo/v4 v4.1.16
	go.opentelemetry.io/contrib v0.0.0 // indirect
	go.opentelemetry.io/contrib/plugins/labstack/echo/internal v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.5.0
)

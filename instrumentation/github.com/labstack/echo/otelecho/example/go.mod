module go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho/example

go 1.20

replace (
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho => ../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

require (
	github.com/labstack/echo/v4 v4.11.4
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.48.0
	go.opentelemetry.io/otel v1.23.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.23.1
	go.opentelemetry.io/otel/sdk v1.23.1
	go.opentelemetry.io/otel/trace v1.23.1
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.23.1 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
)

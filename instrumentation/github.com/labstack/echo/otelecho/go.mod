module go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho

go 1.22.0

toolchain go1.23.4

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3

require (
	github.com/labstack/echo/v4 v4.13.2
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/contrib/propagators/b3 v1.33.0
	go.opentelemetry.io/otel v1.33.0
	go.opentelemetry.io/otel/trace v1.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.33.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

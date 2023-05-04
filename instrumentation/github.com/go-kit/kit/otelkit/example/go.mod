module go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/example

go 1.18

require (
	github.com/gorilla/mux v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit v0.41.1
	go.opentelemetry.io/otel v1.16.0-rc.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.16.0-rc.1
	go.opentelemetry.io/otel/sdk v1.16.0-rc.1
	go.opentelemetry.io/otel/trace v1.16.0-rc.1
)

require (
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.16.0-rc.1 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit => ../

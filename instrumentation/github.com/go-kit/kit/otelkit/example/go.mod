module go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit/example

go 1.18

require (
	github.com/gorilla/mux v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit v0.36.4
	go.opentelemetry.io/otel v1.11.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.11.1
	go.opentelemetry.io/otel/sdk v1.11.1
	go.opentelemetry.io/otel/trace v1.11.1
)

require (
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit => ../

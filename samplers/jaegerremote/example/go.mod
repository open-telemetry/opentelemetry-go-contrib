module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.1.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.7.0
	go.opentelemetry.io/otel/sdk v1.7.0
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

module go.opentelemetry.io/contrib/samplers/jaeger_remote/example

go 1.16

require (
	go.opentelemetry.io/contrib/samplers/jaeger_remote v0.22.0
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
)

replace go.opentelemetry.io/contrib/samplers/jaeger_remote => ../

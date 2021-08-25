module go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/_example

go 1.16

replace go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen => ../

require (
	github.com/99designs/gqlgen v0.13.0
	github.com/vektah/gqlparser/v2 v2.1.0
	go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
)

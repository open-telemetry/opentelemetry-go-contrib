module go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen => ../
)

require (
	github.com/99designs/gqlgen v0.14.0
	github.com/vektah/gqlparser/v2 v2.2.0
	go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen v0.19.0
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC3
	go.opentelemetry.io/otel/sdk v1.0.0-RC3
	go.opentelemetry.io/otel/trace v1.0.0-RC3
)

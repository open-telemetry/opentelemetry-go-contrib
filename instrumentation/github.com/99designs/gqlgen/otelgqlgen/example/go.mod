module go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen => ../
)

require (
	github.com/99designs/gqlgen v0.13.0
	github.com/vektah/gqlparser/v2 v2.1.0
	go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen v0.19.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)

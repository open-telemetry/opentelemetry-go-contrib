module go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen

go 1.15

replace go.opentelemetry.io/contrib => ../../../../../

require (
	github.com/99designs/gqlgen v0.14.0
	github.com/vektah/gqlparser/v2 v2.2.0
	go.opentelemetry.io/contrib v0.22.0
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/trace v1.0.0-RC3
)

module go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen => ../
)

require (
	github.com/99designs/gqlgen v0.13.0
	github.com/stretchr/testify v1.7.0
	github.com/vektah/gqlparser/v2 v2.1.0
	go.opentelemetry.io/contrib v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/oteltest v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
)

module go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo => ../
)

require (
	github.com/google/uuid v1.3.0
	github.com/graph-gophers/graphql-go v1.1.0
	go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
)

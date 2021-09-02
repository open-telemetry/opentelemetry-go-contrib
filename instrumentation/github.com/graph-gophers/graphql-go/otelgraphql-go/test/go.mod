module go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo/test

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo => ../
)

require (
	github.com/graph-gophers/graphql-go v1.1.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/sdk v1.0.0-RC2.0.20210729170058-11f62640ee67
)

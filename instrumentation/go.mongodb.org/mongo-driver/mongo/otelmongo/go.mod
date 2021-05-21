module go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo

go 1.13

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/stretchr/testify v1.7.0
	go.mongodb.org/mongo-driver v1.5.2
	go.opentelemetry.io/contrib v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/oteltest v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)

module go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo

go 1.13

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/stretchr/testify v1.7.0
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.5.0
	go.opentelemetry.io/contrib v0.18.0
	go.opentelemetry.io/otel v0.18.0
	go.opentelemetry.io/otel/oteltest v0.18.0
	go.opentelemetry.io/otel/trace v0.18.0
)

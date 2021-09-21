module go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo

go 1.13

replace go.opentelemetry.io/contrib => ../../../../..

require (
	go.mongodb.org/mongo-driver v1.7.2
	go.opentelemetry.io/otel v1.0.0
	go.opentelemetry.io/otel/trace v1.0.0
)

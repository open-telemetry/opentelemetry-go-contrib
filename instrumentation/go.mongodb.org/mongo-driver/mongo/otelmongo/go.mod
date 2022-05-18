module go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo

go 1.13

replace go.opentelemetry.io/contrib => ../../../../..

require (
	go.mongodb.org/mongo-driver v1.9.1
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
)

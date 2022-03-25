module go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/test

go 1.16

require (
	github.com/stretchr/testify v1.7.1
	go.mongodb.org/mongo-driver v1.8.4
	go.opentelemetry.io/contrib v1.5.0
	go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo v0.30.0
	go.opentelemetry.io/otel v1.6.0
	go.opentelemetry.io/otel/sdk v1.6.0
	go.opentelemetry.io/otel/trace v1.6.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo => ../
)

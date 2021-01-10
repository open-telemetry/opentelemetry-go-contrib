module go.opentelemetry.io/contrib/instrumentation/github.com/streadway/amqp/otelamqp/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/streadway/amqp/otelamqp => ../
)

require (
	github.com/streadway/amqp v1.0.0
	go.opentelemetry.io/contrib/instrumentation/github.com/streadway/amqp/otelamqp v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.15.0
	go.opentelemetry.io/otel/exporters/stdout v0.15.0
	go.opentelemetry.io/otel/sdk v0.15.0
)

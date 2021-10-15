module go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/otelconfluent/example

go 1.15

replace (
	go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/otelconfluent => ../
	go.opentelemetry.io/contrib => ../../../../../../
)

require (
	github.com/confluentinc/confluent-kafka-go v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/otelconfluent v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.1
	go.opentelemetry.io/otel/sdk v1.0.1
)

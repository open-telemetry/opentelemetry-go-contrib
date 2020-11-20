module go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka/example

go 1.13

replace (
	go.opentelemetry.io/contrib => ../../../../../../..
	go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka => ../
)

require (
	github.com/confluentinc/confluent-kafka-go v1.5.2
	go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.13.0
	go.opentelemetry.io/otel/exporters/stdout v0.13.0
	go.opentelemetry.io/otel/sdk v0.13.0
)

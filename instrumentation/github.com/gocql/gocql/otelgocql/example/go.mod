module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../
)

require (
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.29.0
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/exporters/prometheus v0.27.0
	go.opentelemetry.io/otel/exporters/zipkin v1.5.0
	go.opentelemetry.io/otel/metric v0.27.0
	go.opentelemetry.io/otel/sdk v1.5.0
	go.opentelemetry.io/otel/sdk/metric v0.27.0
)

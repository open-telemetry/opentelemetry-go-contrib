module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../
)

require (
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.26.0
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/prometheus v0.24.0
	go.opentelemetry.io/otel/exporters/zipkin v1.1.0
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/sdk/export/metric v0.24.0
	go.opentelemetry.io/otel/sdk/metric v0.24.0
)

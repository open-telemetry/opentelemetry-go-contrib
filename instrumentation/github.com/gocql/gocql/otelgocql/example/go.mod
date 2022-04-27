module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../
)

require (
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.31.0
	go.opentelemetry.io/otel v1.6.3
	go.opentelemetry.io/otel/exporters/prometheus v0.29.0
	go.opentelemetry.io/otel/exporters/zipkin v1.6.3
	go.opentelemetry.io/otel/metric v0.29.0
	go.opentelemetry.io/otel/sdk v1.6.3
	go.opentelemetry.io/otel/sdk/metric v0.29.0
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
)

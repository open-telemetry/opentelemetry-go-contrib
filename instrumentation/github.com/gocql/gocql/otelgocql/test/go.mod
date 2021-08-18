module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/test

go 1.15

require (
	github.com/gocql/gocql v0.0.0-20210707082121-9a3953d1826d
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.22.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.22.0
	go.opentelemetry.io/otel v1.0.0-RC2.0.20210812161231-a8bb0bf89f3b
	go.opentelemetry.io/otel/metric v0.22.0
	go.opentelemetry.io/otel/sdk v1.0.0-RC2.0.20210812161231-a8bb0bf89f3b
	go.opentelemetry.io/otel/trace v1.0.0-RC2
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../

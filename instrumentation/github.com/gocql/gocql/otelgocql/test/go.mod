module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/test

go 1.16

require (
	github.com/gocql/gocql v0.0.0-20210707082121-9a3953d1826d
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib v1.6.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.31.0
	go.opentelemetry.io/otel v1.6.3
	go.opentelemetry.io/otel/sdk v1.6.3
	go.opentelemetry.io/otel/sdk/metric v0.29.1-0.20220425151224-b8e4241a32f2
	go.opentelemetry.io/otel/trace v1.6.3
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../

replace go.opentelemetry.io/contrib => ../../../../../../

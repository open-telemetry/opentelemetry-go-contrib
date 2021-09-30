module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql

go 1.15

replace go.opentelemetry.io/contrib => ../../../../../

require (
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	github.com/golang/snappy v0.0.1 // indirect
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.0.0
	go.opentelemetry.io/otel/internal/metric v0.23.1-0.20210928160814-00d8ca5890a8 // indirect
	go.opentelemetry.io/otel/metric v0.23.1-0.20210928160814-00d8ca5890a8
	go.opentelemetry.io/otel/sdk/metric v0.23.1-0.20210928160814-00d8ca5890a8 // indirect
	go.opentelemetry.io/otel/trace v1.0.0
)

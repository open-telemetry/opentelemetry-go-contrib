module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp

go 1.16

replace go.opentelemetry.io/contrib => ../../../..

require (
	github.com/felixge/httpsnoop v1.0.2
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/otel v1.6.4-0.20220425151224-b8e4241a32f2
	go.opentelemetry.io/otel/metric v0.29.0
	go.opentelemetry.io/otel/trace v1.6.3
)

module go.opentelemetry.io/contrib/instrumentation/net/http

go 1.14

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/felixge/httpsnoop v1.0.1
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.10.1
	go.opentelemetry.io/otel v0.10.0
	google.golang.org/grpc v1.31.0
)

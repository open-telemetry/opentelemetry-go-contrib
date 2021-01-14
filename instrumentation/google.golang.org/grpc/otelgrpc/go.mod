module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc

go 1.14

replace go.opentelemetry.io/contrib => ../../../../

require (
	github.com/golang/protobuf v1.4.3
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.16.0
	go.opentelemetry.io/otel v0.16.0
	google.golang.org/grpc v1.34.0
)

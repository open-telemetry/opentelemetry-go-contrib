module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/test

go 1.16

require (
	github.com/golang/protobuf v1.5.2
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.29.0
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/sdk v1.5.0
	go.uber.org/goleak v1.1.12
	google.golang.org/grpc v1.45.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../
)

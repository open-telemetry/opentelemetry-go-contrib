module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc

go 1.14

replace go.opentelemetry.io/contrib => ../../../../

require (
	github.com/golang/protobuf v1.4.3
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.17.0
	go.opentelemetry.io/otel v0.17.0
	go.opentelemetry.io/otel/oteltest v0.17.0
	go.opentelemetry.io/otel/trace v0.17.0
	go.uber.org/goleak v1.1.10
	google.golang.org/grpc v1.35.0
)

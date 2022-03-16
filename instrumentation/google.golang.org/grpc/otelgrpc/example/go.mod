module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../
)

require (
	github.com/golang/protobuf v1.5.2
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.29.0
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.5.0
	go.opentelemetry.io/otel/sdk v1.5.0
	go.opentelemetry.io/otel/trace v1.5.0
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	google.golang.org/grpc v1.45.0
)

module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/example

go 1.21

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../

require (
	github.com/golang/protobuf v1.5.4
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.50.0
	go.opentelemetry.io/otel v1.25.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.25.0
	go.opentelemetry.io/otel/sdk v1.25.0
	go.opentelemetry.io/otel/trace v1.25.0
	golang.org/x/net v0.24.0
	google.golang.org/grpc v1.63.2
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.25.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

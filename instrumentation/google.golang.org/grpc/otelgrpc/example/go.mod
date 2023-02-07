module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../

require (
	github.com/golang/protobuf v1.5.2
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.38.0
	go.opentelemetry.io/otel v1.13.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.13.0
	go.opentelemetry.io/otel/sdk v1.13.0
	go.opentelemetry.io/otel/trace v1.13.0
	golang.org/x/net v0.5.0
	google.golang.org/grpc v1.52.3
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v0.36.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

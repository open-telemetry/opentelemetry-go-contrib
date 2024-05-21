module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/example

go 1.21

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../

require (
	github.com/golang/protobuf v1.5.4
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.51.0
	go.opentelemetry.io/otel v1.26.1-0.20240521154638-0d3dddc17fcb
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.1-0.20240521154638-0d3dddc17fcb
	go.opentelemetry.io/otel/sdk v1.26.1-0.20240521154638-0d3dddc17fcb
	go.opentelemetry.io/otel/trace v1.26.1-0.20240521154638-0d3dddc17fcb
	golang.org/x/net v0.25.0
	google.golang.org/grpc v1.64.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.26.1-0.20240521154638-0d3dddc17fcb // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240520151616-dc85e6b867a5 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

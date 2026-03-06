module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/example

go 1.25.0

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../

require (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.66.0
	go.opentelemetry.io/otel v1.41.1-0.20260303203755-5deb0d31ed71
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.41.1-0.20260303203755-5deb0d31ed71
	go.opentelemetry.io/otel/sdk v1.41.1-0.20260303203755-5deb0d31ed71
	go.opentelemetry.io/otel/trace v1.41.1-0.20260303203755-5deb0d31ed71
	google.golang.org/grpc v1.79.2
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.41.1-0.20260303203755-5deb0d31ed71 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

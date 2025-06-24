module go.opentelemetry.io/contrib/propagators/opencensus/examples

go 1.23.0

require (
	go.opencensus.io v0.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0
	go.opentelemetry.io/contrib/propagators/opencensus v0.61.0
	go.opentelemetry.io/otel v1.36.1-0.20250624070221-351a51f1ed18
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.36.1-0.20250624070221-351a51f1ed18
	go.opentelemetry.io/otel/sdk v1.36.1-0.20250624070221-351a51f1ed18
	google.golang.org/grpc v1.73.0
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v1.36.1-0.20250624070221-351a51f1ed18 // indirect
	go.opentelemetry.io/otel/metric v1.36.1-0.20250624070221-351a51f1ed18 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.36.1-0.20250624070221-351a51f1ed18 // indirect
	go.opentelemetry.io/otel/trace v1.36.1-0.20250624070221-351a51f1ed18 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../../../instrumentation/google.golang.org/grpc/otelgrpc
	go.opentelemetry.io/contrib/propagators/opencensus => ../
)

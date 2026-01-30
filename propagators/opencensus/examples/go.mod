module go.opentelemetry.io/contrib/propagators/opencensus/examples

go 1.24.0

require (
	go.opencensus.io v0.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.64.0
	go.opentelemetry.io/contrib/propagators/opencensus v0.64.0
	go.opentelemetry.io/otel v1.39.1-0.20260130171517-3264bf171b1e
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.39.1-0.20260130171517-3264bf171b1e
	go.opentelemetry.io/otel/sdk v1.39.1-0.20260130171517-3264bf171b1e
	google.golang.org/grpc v1.78.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v1.39.1-0.20260130171517-3264bf171b1e // indirect
	go.opentelemetry.io/otel/metric v1.39.1-0.20260130171517-3264bf171b1e // indirect
	go.opentelemetry.io/otel/sdk/metric v1.39.1-0.20260130171517-3264bf171b1e // indirect
	go.opentelemetry.io/otel/trace v1.39.1-0.20260130171517-3264bf171b1e // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../../../instrumentation/google.golang.org/grpc/otelgrpc
	go.opentelemetry.io/contrib/propagators/opencensus => ../
)

module go.opentelemetry.io/contrib/propagators/opencensus/examples

go 1.26.0

require (
	go.opencensus.io v0.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.71.0
	go.opentelemetry.io/contrib/propagators/opencensus v0.71.0
	go.opentelemetry.io/otel v1.47.0-rc.1.0.20260924072922-c131dcfc3885
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.0-rc.1.0.20260924072922-c131dcfc3885
	go.opentelemetry.io/otel/sdk v1.47.0-rc.1.0.20260924072922-c131dcfc3885
	google.golang.org/grpc v1.83.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v1.47.0-rc.1.0.20260924072922-c131dcfc3885 // indirect
	go.opentelemetry.io/otel/log v1.47.0-rc.1 // indirect
	go.opentelemetry.io/otel/metric v1.47.0-rc.1.0.20260924072922-c131dcfc3885 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0-rc.1.0.20260924072922-c131dcfc3885 // indirect
	go.opentelemetry.io/otel/trace v1.47.0-rc.1.0.20260924072922-c131dcfc3885 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260921155816-b14227669459 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../../../instrumentation/google.golang.org/grpc/otelgrpc
	go.opentelemetry.io/contrib/propagators/opencensus => ../
)

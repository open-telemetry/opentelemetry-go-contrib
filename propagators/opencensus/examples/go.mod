module go.opentelemetry.io/contrib/propagators/opencensus/examples

go 1.19

require (
	go.opencensus.io v0.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.42.0-rc.1
	go.opentelemetry.io/contrib/propagators/opencensus v0.42.0-rc.1
	go.opentelemetry.io/otel v1.16.0-rc.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.16.0-rc.1
	go.opentelemetry.io/otel/sdk v1.16.0-rc.1
	google.golang.org/grpc v1.55.0
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v0.39.0-rc.1 // indirect
	go.opentelemetry.io/otel/metric v1.16.0-rc.1 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.39.0-rc.1 // indirect
	go.opentelemetry.io/otel/trace v1.16.0-rc.1 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../../../instrumentation/google.golang.org/grpc/otelgrpc
	go.opentelemetry.io/contrib/propagators/opencensus => ../
)

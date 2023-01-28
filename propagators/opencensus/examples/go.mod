module go.opentelemetry.io/contrib/propagators/opencensus/examples

go 1.18

require (
	go.opencensus.io v0.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.37.0
	go.opentelemetry.io/contrib/propagators/opencensus v0.37.0
	go.opentelemetry.io/otel v1.11.3-0.20230126195513-af3db6e8bed6
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.11.2
	go.opentelemetry.io/otel/sdk v1.11.2
	google.golang.org/grpc v1.52.0
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v0.34.0 // indirect
	go.opentelemetry.io/otel/metric v0.34.1-0.20230119184437-b1a8002c4cf5 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.34.1-0.20230119184437-b1a8002c4cf5 // indirect
	go.opentelemetry.io/otel/trace v1.11.2 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../../../instrumentation/google.golang.org/grpc/otelgrpc
	go.opentelemetry.io/contrib/propagators/opencensus => ../
)

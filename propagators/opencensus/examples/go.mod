module go.opentelemetry.io/contrib/propagators/opencensus/examples

go 1.14

require (
	go.opencensus.io v0.22.6-0.20201102222123-380f4078db9f
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.14.0
	go.opentelemetry.io/contrib/propagation/opencensus v0.14.0
	go.opentelemetry.io/otel v0.14.0
	go.opentelemetry.io/otel/exporters/stdout v0.14.0
	go.opentelemetry.io/otel/sdk v0.14.0
	google.golang.org/grpc v1.33.2
)

replace go.opentelemetry.io/contrib/propagation/opencensus => ../

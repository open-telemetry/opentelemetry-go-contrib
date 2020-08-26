module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/example

go 1.14

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc => ../

require (
	github.com/golang/protobuf v1.4.2
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/stdout v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	google.golang.org/grpc v1.31.1
)

module go.opentelemetry.io/contrib/instrumentation/github.com/containerd/ttrpc/otelttrpc

go 1.14

require (
	github.com/containerd/ttrpc v1.0.2
	github.com/gogo/protobuf v1.3.1
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.15.1
	go.opentelemetry.io/otel v0.15.0
	go.opentelemetry.io/otel/exporters/stdout v0.15.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.15.0
	go.opentelemetry.io/otel/sdk v0.15.0
	google.golang.org/grpc v1.31.1
)

module go.opentelemetry.io/contrib/sdk/dynamicconfig

go 1.14

require (
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.7
	github.com/kr/pretty v0.2.0 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.31.0
)

// TODO: delete?
// replace github.com/open-telemetry/opentelemetry-proto => github.com/vmingchen/opentelemetry-proto v0.3.1-0.20200716191220-7eb25882f08b
// replace github.com/open-telemetry/opentelemetry-proto => github.com/vmingchen/opentelemetry-proto v0.3.1-0.20200811161559-2b8579ef37a0
// replace github.com/open-telemetry/opentelemetry-proto => /home/williamtong/go/src/github.com/vmingchen/opentelemetry-proto

replace go.opentelemetry.io/contrib => ../../

replace go.opentelemetry.io/contrib/sdk/dynamicconfig => ./

module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace

go 1.14

replace go.opentelemetry.io/contrib => ../../../..

require (
	github.com/DataDog/sketches-go v0.0.0-20190923095040-43f19ad77ff7 // indirect
	github.com/benbjohnson/clock v1.0.3 // indirect
	github.com/google/go-cmp v0.5.1
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.10.0
	go.opentelemetry.io/otel v0.10.0
	google.golang.org/grpc v1.31.0
)

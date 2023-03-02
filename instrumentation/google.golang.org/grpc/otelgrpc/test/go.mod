module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/test

go 1.18

require (
	github.com/stretchr/testify v1.8.2
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.41.0-rc.1
	go.opentelemetry.io/otel v1.15.0-rc.1
	go.opentelemetry.io/otel/sdk v1.15.0-rc.1
	go.opentelemetry.io/otel/sdk/metric v0.38.0-rc.1
	go.uber.org/goleak v1.2.1
	google.golang.org/grpc v1.53.0
)

require (
	cloud.google.com/go/compute v1.15.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.15.0-rc.1 // indirect
	go.opentelemetry.io/otel/trace v1.15.0-rc.1 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/oauth2 v0.4.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => ../

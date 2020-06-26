module go.opentelemetry.io/contrib/sdk/dynamicconfig/example

go 1.13

replace go.opentelemetry.io/contrib/sdk/dynamicconfig => ../

replace go.opentelemetry.io/otel => github.com/open-telemetry/opentelemetry-go v0.6.1-0.20200623190015-2966505271c3

replace go.opentelemetry.io/otel/exporters/otlp => github.com/open-telemetry/opentelemetry-go/exporters/otlp v0.6.1-0.20200623190015-2966505271c3

require (
	github.com/open-telemetry/opentelemetry-proto v0.3.0 // indirect
	github.com/vmingchen/opentelemetry-proto v0.3.1-0.20200611154326-5406581153f7
	go.opentelemetry.io/contrib/sdk/dynamicconfig v0.6.1
	go.opentelemetry.io/otel v0.6.0
	go.opentelemetry.io/otel/exporters/otlp v0.6.0
	google.golang.org/grpc v1.30.0
)

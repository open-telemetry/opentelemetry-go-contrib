module go.opentelemetry.io/contrib/sdk/dynamicconfig/example

go 1.13

replace github.com/open-telemetry/opentelemetry-proto => github.com/vmingchen/opentelemetry-proto v0.3.1-0.20200707164106-b68642716098

replace go.opentelemetry.io/contrib/sdk/dynamicconfig => ../

require (
	github.com/open-telemetry/opentelemetry-proto v0.4.0
	go.opentelemetry.io/contrib/sdk/dynamicconfig v0.6.1
	go.opentelemetry.io/otel v0.7.0
	go.opentelemetry.io/otel/exporters/otlp v0.7.0
	google.golang.org/grpc v1.30.0
)

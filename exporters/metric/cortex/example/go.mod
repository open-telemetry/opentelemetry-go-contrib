module go.opentelemetry.io/contrib/exporters/metric/cortex/example

go 1.14

replace (
	go.opentelemetry.io/contrib/exporters/metric/cortex => ../
	go.opentelemetry.io/contrib/exporters/metric/cortex/utils => ../utils/
)

require (
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.19.0
	go.opentelemetry.io/contrib/exporters/metric/cortex/utils v0.19.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/metric v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/sdk/metric v0.19.0
	gopkg.in/yaml.v2 v2.2.5 // indirect
)

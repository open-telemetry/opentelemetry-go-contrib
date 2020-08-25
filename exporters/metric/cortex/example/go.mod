module go.opentelemetry.io/contrib/exporters/metric/cortex/example

go 1.14

replace (
	go.opentelemetry.io/contrib/exporters/metric/cortex => ../
	go.opentelemetry.io/contrib/exporters/metric/cortex/utils => ../utils
)

require (
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.11.0
	go.opentelemetry.io/contrib/exporters/metric/cortex/utils v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
	gopkg.in/yaml.v2 v2.2.5 // indirect
)

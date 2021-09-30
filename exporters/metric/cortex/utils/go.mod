module go.opentelemetry.io/contrib/exporters/metric/cortex/utils

go 1.15

replace go.opentelemetry.io/contrib/exporters/metric/cortex => ../

require (
	github.com/spf13/afero v1.6.0
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.23.0
	go.opentelemetry.io/otel/metric v0.23.1-0.20210928160814-00d8ca5890a8 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.23.1-0.20210928160814-00d8ca5890a8 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.23.1-0.20210928160814-00d8ca5890a8 // indirect
)

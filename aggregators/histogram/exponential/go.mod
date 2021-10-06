module go.opentelemetry.io/contrib/aggregators/histogram/exponential

go 1.16

require (
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/sdk/export/metric v0.24.0
	go.opentelemetry.io/otel/sdk/metric v0.24.0
)

replace go.opentelemetry.io/otel/sdk/metric => ../../../../go/sdk/metric
replace go.opentelemetry.io/otel/sdk/export/metric => ../../../../go/sdk/export/metric
replace go.opentelemetry.io/otel/metric => ../../../../go/metric

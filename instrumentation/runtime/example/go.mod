module go.opentelemetry.io/contrib/instrumentation/runtime/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/runtime => ../

require (
	go.opentelemetry.io/contrib/instrumentation/runtime v0.36.2
	go.opentelemetry.io/otel v1.11.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.32.3
	go.opentelemetry.io/otel/metric v0.32.3
	go.opentelemetry.io/otel/sdk v1.11.0
	go.opentelemetry.io/otel/sdk/metric v0.32.3
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/trace v1.11.0 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

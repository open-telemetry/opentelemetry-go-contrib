module go.opentelemetry.io/contrib/propagators/autoprop

go 1.18

require (
	github.com/stretchr/testify v1.8.1
	go.opentelemetry.io/contrib/propagators/aws v1.11.1
	go.opentelemetry.io/contrib/propagators/b3 v1.11.1
	go.opentelemetry.io/contrib/propagators/jaeger v1.11.1
	go.opentelemetry.io/contrib/propagators/ot v1.11.1
	go.opentelemetry.io/otel v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/sdk v1.11.1 // indirect
	go.opentelemetry.io/otel/trace v1.11.1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/propagators/jaeger => ../jaeger

replace go.opentelemetry.io/contrib/propagators/b3 => ../b3

replace go.opentelemetry.io/contrib/propagators/aws => ../aws

replace go.opentelemetry.io/contrib/propagators/ot => ../ot

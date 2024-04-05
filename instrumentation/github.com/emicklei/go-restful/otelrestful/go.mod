module go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful

go 1.21

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3

require (
	github.com/emicklei/go-restful/v3 v3.12.0
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/propagators/b3 v1.24.0
	go.opentelemetry.io/otel v1.25.0
	go.opentelemetry.io/otel/sdk v1.25.0
	go.opentelemetry.io/otel/trace v1.25.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.25.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

module go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful

go 1.18

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3

require (
	github.com/emicklei/go-restful/v3 v3.9.0
	github.com/stretchr/testify v1.8.0
	go.opentelemetry.io/contrib/propagators/b3 v1.10.0
	go.opentelemetry.io/otel v1.11.0
	go.opentelemetry.io/otel/trace v1.11.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

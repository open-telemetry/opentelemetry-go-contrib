module go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	github.com/emicklei/go-restful/v3 v3.3.1
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.14.0
	go.opentelemetry.io/contrib/propagators v0.14.0
	go.opentelemetry.io/otel v0.14.0
)

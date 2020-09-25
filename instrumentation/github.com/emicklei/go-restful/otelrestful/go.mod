module go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/emicklei/go-restful/v3 v3.3.1
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.12.0
	go.opentelemetry.io/contrib/propagators v0.0.0-20200924185937-b313ddb2989e
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/stdout v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
)

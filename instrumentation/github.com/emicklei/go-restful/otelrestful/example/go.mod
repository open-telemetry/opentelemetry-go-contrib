module go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful/example

go 1.21

replace (
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful => ../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

require (
	github.com/emicklei/go-restful/v3 v3.12.0
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful v0.52.0
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.27.0
	go.opentelemetry.io/otel/sdk v1.27.0
	go.opentelemetry.io/otel/trace v1.27.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.27.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

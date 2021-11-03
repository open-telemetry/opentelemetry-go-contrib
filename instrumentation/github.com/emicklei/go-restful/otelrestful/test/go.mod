module go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful/test

go 1.15

require (
	github.com/emicklei/go-restful/v3 v3.7.1
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful v0.26.0
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/trace v1.1.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful => ../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

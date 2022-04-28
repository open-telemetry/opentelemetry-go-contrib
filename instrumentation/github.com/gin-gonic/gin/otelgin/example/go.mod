module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin/example

go 1.16

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin => ../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

require (
	github.com/gin-gonic/gin v1.7.7
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.31.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.7.0
	go.opentelemetry.io/otel/sdk v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
)

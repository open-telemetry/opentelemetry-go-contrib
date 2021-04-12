module go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/gin-gonic/gin/otelgin/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin => ../
	go.opentelemetry.io/contrib/propagators => ../../../../../../propagators
)

require (
	github.com/gin-gonic/gin v1.7.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.19.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
)

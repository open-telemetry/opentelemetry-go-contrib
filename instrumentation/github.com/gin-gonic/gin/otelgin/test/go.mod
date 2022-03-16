module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin/test

go 1.16

require (
	github.com/gin-gonic/gin v1.7.7
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.29.0
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/sdk v1.5.0
	go.opentelemetry.io/otel/trace v1.5.0
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin => ../

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3

replace go.opentelemetry.io/contrib => ../../../../../../

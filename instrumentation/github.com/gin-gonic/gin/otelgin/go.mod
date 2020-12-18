module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.15.1
	go.opentelemetry.io/contrib/propagators v0.15.1
	go.opentelemetry.io/otel v0.15.0
)

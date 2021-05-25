module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	github.com/gin-gonic/gin v1.7.2
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.20.0
	go.opentelemetry.io/contrib/propagators v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/oteltest v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)

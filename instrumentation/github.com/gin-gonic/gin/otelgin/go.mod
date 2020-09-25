module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../propagators/b3
)

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.12.0
	go.opentelemetry.io/contrib/propagators v0.0.0-20200924185937-b313ddb2989e
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/stdout v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
)

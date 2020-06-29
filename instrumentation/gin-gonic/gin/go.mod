module go.opentelemetry.io/contrib/instrumentation/gin-gonic/gin

go 1.14

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.6.1
	go.opentelemetry.io/otel v0.7.0
	google.golang.org/grpc v1.30.0
)

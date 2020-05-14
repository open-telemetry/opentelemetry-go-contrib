module go.opentelemetry.io/contrib/plugins/gin-gonic/gin

go 1.14

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/gin-gonic/gin v1.6.2
	github.com/stretchr/testify v1.4.0
	go.opentelemetry.io/contrib v0.0.0-20200417154017-e4eb804471ca
	go.opentelemetry.io/otel v0.5.0
	google.golang.org/grpc v1.28.1
)

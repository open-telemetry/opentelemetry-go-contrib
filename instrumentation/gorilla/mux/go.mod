module go.opentelemetry.io/contrib/instrumentation/gorilla/mux

go 1.14

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/gorilla/mux v1.7.4
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.6.1
	go.opentelemetry.io/otel v0.6.0
)

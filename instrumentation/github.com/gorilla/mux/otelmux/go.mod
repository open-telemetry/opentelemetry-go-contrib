module go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux

go 1.15

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/felixge/httpsnoop v1.0.2
	github.com/gorilla/mux v1.8.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/trace v1.0.1
)

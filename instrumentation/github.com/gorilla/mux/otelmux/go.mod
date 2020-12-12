module go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../..
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	github.com/felixge/httpsnoop v1.0.1
	github.com/gorilla/mux v1.8.0
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.15.0
	go.opentelemetry.io/contrib/propagators v0.15.0
	go.opentelemetry.io/otel v0.15.0
)

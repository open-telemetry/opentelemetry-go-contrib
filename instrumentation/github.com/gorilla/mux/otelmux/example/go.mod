module go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux => ../

require (
	github.com/gorilla/mux v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.41.1
	go.opentelemetry.io/otel v1.15.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.15.1
	go.opentelemetry.io/otel/sdk v1.15.1
	go.opentelemetry.io/otel/trace v1.15.1
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

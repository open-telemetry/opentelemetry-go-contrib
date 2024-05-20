module go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/example

go 1.21

replace go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux => ../

require (
	github.com/gorilla/mux v1.8.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.51.0
	go.opentelemetry.io/otel v1.26.1-0.20240519051633-999c6a07b318
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.1-0.20240519051633-999c6a07b318
	go.opentelemetry.io/otel/sdk v1.26.1-0.20240519051633-999c6a07b318
	go.opentelemetry.io/otel/trace v1.26.1-0.20240519051633-999c6a07b318
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v1.26.1-0.20240519051633-999c6a07b318 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

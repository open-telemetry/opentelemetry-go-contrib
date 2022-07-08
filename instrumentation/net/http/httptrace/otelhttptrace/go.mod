module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace

go 1.17

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/google/go-cmp v0.5.8
	go.opentelemetry.io/otel v1.8.0
	go.opentelemetry.io/otel/trace v1.8.0
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
)

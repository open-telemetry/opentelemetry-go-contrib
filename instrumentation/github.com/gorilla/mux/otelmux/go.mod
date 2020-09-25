module go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/gorilla/mux v1.8.0
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.12.0
	go.opentelemetry.io/contrib/propagators v0.0.0-20200924185937-b313ddb2989e
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/stdout v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
)

module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/test

go 1.16

require (
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.29.0
	go.opentelemetry.io/otel v1.4.1
	go.opentelemetry.io/otel/sdk v1.4.1
)

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace => ../
)

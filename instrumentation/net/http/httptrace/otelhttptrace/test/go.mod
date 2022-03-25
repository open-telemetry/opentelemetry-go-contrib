module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/test

go 1.16

require (
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.30.0
	go.opentelemetry.io/otel v1.6.0
	go.opentelemetry.io/otel/sdk v1.6.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace => ../
)

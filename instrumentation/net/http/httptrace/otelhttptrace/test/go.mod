module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/test

go 1.16

require (
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.31.0
	go.opentelemetry.io/otel v1.6.4-0.20220425151224-b8e4241a32f2
	go.opentelemetry.io/otel/sdk v1.6.3
)

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace => ../
)

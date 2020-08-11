module go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego

go 1.15

require (
	github.com/astaxie/beego v1.12.2
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.10.0
	go.opentelemetry.io/contrib/instrumentation/net/http v0.0.0-20200806162034-3fc65dc78f63
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
)

replace go.opentelemetry.io/contrib => ../../../..

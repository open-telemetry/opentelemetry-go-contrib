module go.opentelemetry.io/contrib/instrumentation/astaxie/beego/otelbeego/example

go 1.14

require (
	github.com/astaxie/beego v1.12.2
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego v0.11.0
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/stdout v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego => ../
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../../net/http/otelhttp
)

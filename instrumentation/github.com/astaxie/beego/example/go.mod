module go.opentelemetry.io/contrib/instrumentation/astaxie/beego/example

go 1.14

require (
	github.com/astaxie/beego v1.12.2
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/stdout v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego => ../
)

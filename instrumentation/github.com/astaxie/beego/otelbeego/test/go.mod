module go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/test

go 1.15

require (
	github.com/astaxie/beego v1.12.3
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego v0.22.0
	go.opentelemetry.io/contrib/propagators/b3 v0.22.0
	go.opentelemetry.io/otel v1.0.0-RC2.0.20210812161231-a8bb0bf89f3b
	go.opentelemetry.io/otel/metric v0.22.0
	go.opentelemetry.io/otel/sdk v1.0.0-RC2.0.20210812161231-a8bb0bf89f3b
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego => ../

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3

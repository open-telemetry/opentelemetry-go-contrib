module go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/test

go 1.15

require (
	github.com/astaxie/beego v1.12.3
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego v0.22.0
	go.opentelemetry.io/contrib/propagators/b3 v0.22.0
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/metric v0.22.0
	go.opentelemetry.io/otel/sdk v1.0.0-RC3
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego => ../

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3

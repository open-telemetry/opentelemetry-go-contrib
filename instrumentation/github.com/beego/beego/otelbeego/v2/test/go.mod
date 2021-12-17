module go.opentelemetry.io/contrib/instrumentation/github.com/beego/beego/otelbeego/v2/test

go 1.16

require (
	github.com/beego/beego/v2 v2.0.1
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/beego/beego/otelbeego/v2 v0.27.0
	go.opentelemetry.io/contrib/propagators/b3 v1.2.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/metric v0.26.0
	go.opentelemetry.io/otel/sdk v1.3.0
)

replace (
	go.opentelemetry.io/contrib => ../../../../../../
	go.opentelemetry.io/contrib/instrumentation/github.com/beego/beego/otelbeego/v2 => ../
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../../net/http/otelhttp
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

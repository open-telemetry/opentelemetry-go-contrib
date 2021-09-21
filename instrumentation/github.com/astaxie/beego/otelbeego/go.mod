module go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../net/http/otelhttp
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3
)

require (
	github.com/astaxie/beego v1.12.3
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.24.0
	go.opentelemetry.io/otel v1.0.0
	go.opentelemetry.io/otel/metric v0.23.0
	go.opentelemetry.io/otel/trace v1.0.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200803210538-64077c9b5642 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)

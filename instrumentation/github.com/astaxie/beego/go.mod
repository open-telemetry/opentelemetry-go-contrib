module go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego

go 1.14

require (
	github.com/astaxie/beego v1.12.2
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.10.1
	go.opentelemetry.io/contrib/instrumentation/net/http v0.10.1
	go.opentelemetry.io/otel v0.10.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200803210538-64077c9b5642 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200804131852-c06518451d9c // indirect
	google.golang.org/grpc v1.31.0
	google.golang.org/protobuf v1.25.0 // indirect
)

replace go.opentelemetry.io/contrib => ../../../..

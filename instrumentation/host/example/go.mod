module go.opentelemetry.io/contrib/instrumentation/host/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/host => ../

require (
	go.opentelemetry.io/contrib/instrumentation/host v0.37.0
	go.opentelemetry.io/otel v1.12.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.35.0
	go.opentelemetry.io/otel/sdk v1.12.0
	go.opentelemetry.io/otel/sdk/metric v0.35.0
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shirou/gopsutil/v3 v3.22.12 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opentelemetry.io/otel/metric v0.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.12.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
)

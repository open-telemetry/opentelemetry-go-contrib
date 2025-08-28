module go.opentelemetry.io/contrib/instrumentation/host/example

go 1.23.0

replace go.opentelemetry.io/contrib/instrumentation/host => ../

require (
	go.opentelemetry.io/contrib/instrumentation/host v0.62.0
	go.opentelemetry.io/otel v1.37.1-0.20250828092952-5358fd737d0c
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.37.1-0.20250825143334-4b2bef6dd972
	go.opentelemetry.io/otel/sdk v1.37.1-0.20250825143334-4b2bef6dd972
	go.opentelemetry.io/otel/sdk/metric v1.37.1-0.20250825143334-4b2bef6dd972
)

require (
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20250827001030-24949be3fa54 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/shirou/gopsutil/v4 v4.25.7 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.1-0.20250825143334-4b2bef6dd972 // indirect
	go.opentelemetry.io/otel/trace v1.37.1-0.20250825143334-4b2bef6dd972 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

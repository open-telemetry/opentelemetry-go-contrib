module go.opentelemetry.io/contrib/instrumentation/host

go 1.14

replace go.opentelemetry.io/contrib => ../..

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.17.0
	go.opentelemetry.io/otel v0.17.0
	go.opentelemetry.io/otel/metric v0.17.0
	go.opentelemetry.io/otel/oteltest v0.17.0
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
)

module go.opentelemetry.io/contrib/instrumentation/host

go 1.15

replace go.opentelemetry.io/contrib => ../..

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.0.0
	go.opentelemetry.io/otel/metric v0.23.0
	golang.org/x/sys v0.0.0-20210303074136-134d130e1a04 // indirect
)

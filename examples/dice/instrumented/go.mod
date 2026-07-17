module go.opentelemetry.io/contrib/examples/dice/instrumented

go 1.25.0

require (
	go.opentelemetry.io/contrib/bridges/otelslog v0.19.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0
	go.opentelemetry.io/otel v1.44.1-0.20260713231842-9778affd47e2
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.20.1-0.20260623111333-65f30a1ab958
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.44.1-0.20260623111333-65f30a1ab958
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.1-0.20260623111333-65f30a1ab958
	go.opentelemetry.io/otel/log v0.20.1-0.20260625150014-c84013202f01
	go.opentelemetry.io/otel/metric v1.44.1-0.20260625150014-c84013202f01
	go.opentelemetry.io/otel/sdk v1.44.1-0.20260625150014-c84013202f01
	go.opentelemetry.io/otel/sdk/log v0.20.1-0.20260623111333-65f30a1ab958
	go.opentelemetry.io/otel/sdk/metric v1.44.1-0.20260625150014-c84013202f01
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/trace v1.44.1-0.20260625150014-c84013202f01 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace (
	go.opentelemetry.io/contrib/bridges/otelslog => ../../../bridges/otelslog
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../instrumentation/net/http/otelhttp
)

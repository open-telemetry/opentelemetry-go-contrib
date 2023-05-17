module go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache => ../

require (
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b
	go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache v0.42.0-rc.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.16.0-rc.1
	go.opentelemetry.io/otel/sdk v1.16.0-rc.1
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel v1.16.0-rc.1 // indirect
	go.opentelemetry.io/otel/metric v1.16.0-rc.1 // indirect
	go.opentelemetry.io/otel/trace v1.16.0-rc.1 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

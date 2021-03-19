module go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache

go 1.14

replace go.opentelemetry.io/contrib => ../../../../../../

require (
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.19.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/oteltest v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
)

module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/example

go 1.18

replace go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../

require (
	github.com/gocql/gocql v1.4.0
	github.com/prometheus/client_golang v1.15.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.41.1
	go.opentelemetry.io/otel v1.15.1
	go.opentelemetry.io/otel/exporters/prometheus v0.38.1
	go.opentelemetry.io/otel/exporters/zipkin v1.15.1
	go.opentelemetry.io/otel/metric v0.38.1
	go.opentelemetry.io/otel/sdk v1.15.1
	go.opentelemetry.io/otel/sdk/metric v0.38.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	go.opentelemetry.io/otel/trace v1.15.1 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

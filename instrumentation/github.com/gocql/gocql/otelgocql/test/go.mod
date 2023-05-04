module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/test

go 1.19

require (
	github.com/gocql/gocql v1.4.0
	github.com/stretchr/testify v1.8.2
	go.opentelemetry.io/contrib v1.17.0-rc.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.42.0-rc.1
	go.opentelemetry.io/otel v1.16.0-rc.1
	go.opentelemetry.io/otel/sdk v1.16.0-rc.1
	go.opentelemetry.io/otel/sdk/metric v0.39.0-rc.1
	go.opentelemetry.io/otel/trace v1.16.0-rc.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0-rc.1 // indirect
	golang.org/x/sys v0.7.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../

replace go.opentelemetry.io/contrib => ../../../../../../

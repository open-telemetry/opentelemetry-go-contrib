module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/test

go 1.18

require (
	github.com/gocql/gocql v0.0.0-20210707082121-9a3953d1826d
	github.com/stretchr/testify v1.8.1
	go.opentelemetry.io/contrib v1.11.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql v0.36.4
	go.opentelemetry.io/otel v1.11.1
	go.opentelemetry.io/otel/sdk v1.11.1
	go.opentelemetry.io/otel/trace v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v0.33.0 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql => ../

replace go.opentelemetry.io/contrib => ../../../../../../

module go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/test

go 1.19

require (
	github.com/astaxie/beego v1.12.3
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego v0.42.0
	go.opentelemetry.io/contrib/propagators/b3 v1.17.0
	go.opentelemetry.io/otel v1.16.0
	go.opentelemetry.io/otel/sdk v1.16.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/shiena/ansicolor v0.0.0-20151119151921-a422bbe96644 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego => ../
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../../net/http/otelhttp
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3
)

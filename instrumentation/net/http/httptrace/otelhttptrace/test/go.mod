module go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.57.0
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/sdk v1.32.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace => ../

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../otelhttp

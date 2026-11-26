// Deprecated: OpenTracing is deprecated.
// v1.48.0 is the final planned release of this module.
// Migrate to native OpenTelemetry APIs and SDKs with W3C Trace Context and W3C Baggage propagation.
module go.opentelemetry.io/contrib/propagators/ot

go 1.26.0

require (
	github.com/google/go-cmp v0.7.0
	github.com/stretchr/testify v1.12.1
	go.opentelemetry.io/otel v1.47.1-0.20261123090150-69dad8df23be
	go.opentelemetry.io/otel/trace v1.47.1-0.20261119094326-34915e1fc1ec
	go.uber.org/multierr v1.11.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.2 // indirect
	go.opentelemetry.io/otel/log v1.47.1-0.20261119094326-34915e1fc1ec // indirect
	go.opentelemetry.io/otel/metric v1.47.1-0.20261119094326-34915e1fc1ec // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

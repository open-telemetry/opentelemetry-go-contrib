module go.opentelemetry.io/contrib/samplers/probability/traceidratio

go 1.25.0

// Replace directives for opentelemetry-go commit with IsRandom/WithRandom (PR #8012).
// Remove when using a released version of opentelemetry-go that includes these APIs.
replace (
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v0.0.0-20260313082256-2ffde5a4289b
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v0.0.0-20260313082256-2ffde5a4289b
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v0.0.0-20260313082256-2ffde5a4289b
	go.opentelemetry.io/otel/sdk/metric => go.opentelemetry.io/otel/sdk/metric v0.0.0-20260313082256-2ffde5a4289b
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v0.0.0-20260313082256-2ffde5a4289b
)

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.42.0
	go.opentelemetry.io/otel/sdk v1.42.0
	go.opentelemetry.io/otel/trace v1.42.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

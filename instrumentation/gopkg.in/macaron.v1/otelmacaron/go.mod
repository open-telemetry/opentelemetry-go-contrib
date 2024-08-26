// Deprecated: otelmacaron has no Code Owner.
module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron

go 1.22

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../propagators/b3

require (
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/propagators/b3 v1.29.0
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/trace v1.29.0
	gopkg.in/macaron.v1 v1.5.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-macaron/inject v0.0.0-20200308113650-138e5925c53b // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/unknwon/com v1.0.1 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

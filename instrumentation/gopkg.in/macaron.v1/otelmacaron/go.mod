module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron

go 1.21

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../propagators/b3

require (
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/propagators/b3 v1.26.0
	go.opentelemetry.io/otel v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/trace v1.26.1-0.20240520052501-49c866fbcd20
	gopkg.in/macaron.v1 v1.5.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-macaron/inject v0.0.0-20200308113650-138e5925c53b // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/unknwon/com v1.0.1 // indirect
	go.opentelemetry.io/otel/metric v1.26.1-0.20240520052501-49c866fbcd20 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

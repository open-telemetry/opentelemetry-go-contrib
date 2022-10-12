module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron

go 1.18

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../propagators/b3

require (
	github.com/stretchr/testify v1.8.0
	go.opentelemetry.io/contrib/propagators/b3 v1.10.0
	go.opentelemetry.io/otel v1.11.0
	go.opentelemetry.io/otel/trace v1.11.0
	gopkg.in/macaron.v1 v1.4.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-macaron/inject v0.0.0-20160627170012-d8a0b8677191 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/unknwon/com v0.0.0-20190804042917-757f69c95f3e // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	gopkg.in/ini.v1 v1.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

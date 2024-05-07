module go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron/example

go 1.21

replace (
	go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron => ../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3
)

require (
	go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron v0.51.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.0
	go.opentelemetry.io/otel/sdk v1.26.0
	go.opentelemetry.io/otel/trace v1.26.0
	gopkg.in/macaron.v1 v1.5.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-macaron/inject v0.0.0-20200308113650-138e5925c53b // indirect
	github.com/unknwon/com v0.0.0-20190804042917-757f69c95f3e // indirect
	go.opentelemetry.io/otel/metric v1.26.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	gopkg.in/ini.v1 v1.66.6 // indirect
)

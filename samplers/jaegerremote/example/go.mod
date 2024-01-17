module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.20

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.15.1
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.22.0
	go.opentelemetry.io/otel/sdk v1.22.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/otel/metric v1.22.0 // indirect
	go.opentelemetry.io/otel/trace v1.22.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	google.golang.org/genproto v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230526203410-71b5a4ffd15e // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

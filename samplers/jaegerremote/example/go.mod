module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.18

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.10.0-rc.1
	go.opentelemetry.io/otel v1.15.0-rc.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.15.0-rc.1
	go.opentelemetry.io/otel/sdk v1.15.0-rc.1
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/otel/metric v1.15.0-rc.1 // indirect
	go.opentelemetry.io/otel/trace v1.15.0-rc.1 // indirect
	golang.org/x/sys v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

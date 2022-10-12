module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.18

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.4.0
	go.opentelemetry.io/otel v1.11.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.11.0
	go.opentelemetry.io/otel/sdk v1.11.0
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/otel/trace v1.11.0 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

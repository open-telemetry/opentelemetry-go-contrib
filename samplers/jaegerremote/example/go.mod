module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.21

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.19.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.0
	go.opentelemetry.io/otel/sdk v1.26.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/otel/metric v1.26.0 // indirect
	go.opentelemetry.io/otel/trace v1.26.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240429193739-8cf5692501f6 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

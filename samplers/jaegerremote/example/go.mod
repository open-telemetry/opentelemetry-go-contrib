module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.21

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.23.0
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.29.0
	go.opentelemetry.io/otel/sdk v1.29.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240823204242-4ba0660f739c // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

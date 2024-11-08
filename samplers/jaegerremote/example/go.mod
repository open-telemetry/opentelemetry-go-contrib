module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.22

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.25.0
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.31.0
	go.opentelemetry.io/otel/sdk v1.32.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241104194629-dd2ea8efbc28 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

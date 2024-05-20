module go.opentelemetry.io/contrib/samplers/jaegerremote/example

go 1.21

require (
	github.com/davecgh/go-spew v1.1.1
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.20.0
	go.opentelemetry.io/otel v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.1-0.20240520052501-49c866fbcd20
	go.opentelemetry.io/otel/sdk v1.26.1-0.20240520052501-49c866fbcd20
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/otel/metric v1.26.1-0.20240520052501-49c866fbcd20 // indirect
	go.opentelemetry.io/otel/trace v1.26.1-0.20240520052501-49c866fbcd20 // indirect
	golang.org/x/sys v0.20.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240515191416-fc5f0ca64291 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace go.opentelemetry.io/contrib/samplers/jaegerremote => ../

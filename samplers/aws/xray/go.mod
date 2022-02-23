module go.opentelemetry.io/contrib/samplers/aws/xray

go 1.16

replace go.opentelemetry.io/contrib/samplers/aws/xray/internal => ../internal

require (
	github.com/go-logr/logr v1.2.2
	github.com/go-logr/stdr v1.2.2
	github.com/jinzhu/copier v0.3.5
	go.opentelemetry.io/otel/sdk v1.4.0
	go.opentelemetry.io/otel/trace v1.4.0
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
)
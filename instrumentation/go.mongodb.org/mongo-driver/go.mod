module go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver

go 1.13

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/stretchr/testify v1.6.1
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.3.5
	go.opentelemetry.io/contrib v0.7.0
	go.opentelemetry.io/otel v0.7.0
	golang.org/x/crypto v0.0.0-20191105034135-c7e5f84aec59 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
)

module go.opentelemetry.io/contrib/go.mongodb.org/mongo-driver/mongo

go 1.13

replace go.opentelemetry.io/contrib => ../../..

require (
	github.com/stretchr/testify v1.4.0
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.3.2
	go.opentelemetry.io/contrib v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/contrib/plugins/gorilla/mux v0.0.0-20200514221819-87b1c6938aeb // indirect
	go.opentelemetry.io/otel v0.5.0
	golang.org/x/crypto v0.0.0-20191105034135-c7e5f84aec59 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
)

replace go.opentelemetry.io/contrib/internal/ext => ../../../internal/ext

replace go.opentelemetry.io/contrib/internal/mocktracer => ../../../internal/mocktracer

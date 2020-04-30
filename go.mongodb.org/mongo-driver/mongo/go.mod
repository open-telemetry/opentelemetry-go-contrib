module go.opentelemetry.io/contrib/go.mongodb.org/mongo-driver/mongo

go 1.13

require (
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.3.2
	go.opentelemetry.io/otel v0.4.3
	golang.org/x/crypto v0.0.0-20191105034135-c7e5f84aec59 // indirect
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	honnef.co/go/tools v0.0.0-20190523083050-ea95bdfd59fc // indirect
)

replace go.opentelemetry.io/contrib/internal/ext => ../../../internal/ext

replace go.opentelemetry.io/contrib/internal/mocktracer => ../../../internal/mocktracer

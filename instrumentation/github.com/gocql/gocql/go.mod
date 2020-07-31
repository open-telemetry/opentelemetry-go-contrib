module go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql

go 1.14

replace go.opentelemetry.io/contrib => ../../../../

require (
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	github.com/golang/snappy v0.0.1 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.10.0
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
	google.golang.org/genproto v0.0.0-20200331122359-1ee6d9798940 // indirect
)

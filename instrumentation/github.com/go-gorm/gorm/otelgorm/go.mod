module go.opentelemetry.io/contrib/instrumentation/github.com/go-gorm/gorm/otelgorm

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC1
	go.opentelemetry.io/otel/oteltest v1.0.0-RC1
	go.opentelemetry.io/otel/sdk v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC1
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.11
)

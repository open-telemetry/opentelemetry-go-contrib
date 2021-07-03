module go.opentelemetry.io/contrib/instrumentation/github.com/go-gorm/gorm/otelgorm

go 1.14

replace go.opentelemetry.io/contrib => ../../../../..

require (
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.21.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/oteltest v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.20.12
)

module go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql/example

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql => ../
)

require (
	github.com/go-sql-driver/mysql v1.5.0
	go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql v0.16.0
	go.opentelemetry.io/otel v0.16.0
	go.opentelemetry.io/otel/exporters/stdout v0.16.0
	go.opentelemetry.io/otel/sdk v0.16.0
)

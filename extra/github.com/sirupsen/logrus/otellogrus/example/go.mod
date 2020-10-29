module go.opentelemetry.io/contrib/extra/github.com/sirupsen/logrus/otellogrus/example

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../..
	go.opentelemetry.io/contrib/extra/github.com/sirupsen/logrus/otellogrus => ../
)

require (
	github.com/sirupsen/logrus v1.7.0
	go.opentelemetry.io/contrib/extra/github.com/sirupsen/logrus/otellogrus v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.13.0
	go.opentelemetry.io/otel/exporters/stdout v0.13.0
	go.opentelemetry.io/otel/sdk v0.13.0
)

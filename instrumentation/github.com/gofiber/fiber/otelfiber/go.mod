module go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/otelfiber

go 1.15

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators/b3 => ../../../../../propagators/b3
)

require (
	github.com/gofiber/fiber/v2 v2.14.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.23.0
	go.opentelemetry.io/contrib/propagators/b3 v0.23.0
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/oteltest v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC3
)

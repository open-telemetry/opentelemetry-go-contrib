module go.opentelemetry.io/contrib/exporters/autoexport

go 1.22.7

require (
	github.com/prometheus/client_golang v1.20.5
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/contrib/bridges/prometheus v0.58.0
	go.opentelemetry.io/otel v1.33.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.9.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.9.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.33.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.33.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.33.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.33.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.33.0
	go.opentelemetry.io/otel/exporters/prometheus v0.55.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.9.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.33.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.33.0
	go.opentelemetry.io/otel/sdk v1.33.0
	go.opentelemetry.io/otel/sdk/log v0.9.0
	go.opentelemetry.io/otel/sdk/metric v1.33.0
	go.opentelemetry.io/proto/otlp v1.4.0
	go.uber.org/goleak v1.3.0
	google.golang.org/protobuf v1.36.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.24.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.61.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/log v0.9.0 // indirect
	go.opentelemetry.io/otel/metric v1.33.0 // indirect
	go.opentelemetry.io/otel/trace v1.33.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241216192217-9240e9c98484 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241216192217-9240e9c98484 // indirect
	google.golang.org/grpc v1.69.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/bridges/prometheus => ../../bridges/prometheus

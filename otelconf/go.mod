module go.opentelemetry.io/contrib/otelconf

go 1.25.0

require (
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/otlptranslator v1.0.0
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/contrib/detectors/aws/ecs v1.43.0
	go.opentelemetry.io/contrib/propagators/autoprop v0.68.0
	go.opentelemetry.io/otel v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/prometheus v0.65.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.19.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/log v0.19.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/metric v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/sdk v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/sdk/log v0.19.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/sdk/log/logtest v0.19.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/sdk/metric v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/otel/trace v1.43.1-0.20260521080857-e5bdc311108b
	go.opentelemetry.io/proto/otlp v1.10.0
	go.yaml.in/yaml/v3 v3.0.4
	golang.org/x/exp v0.0.0-20260527015227-08cc5374adb3
	google.golang.org/grpc v1.81.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/brunoscheufler/aws-ecs-metadata-go v0.0.0-20221221133751-67e37ae746cd // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/propagators/aws v1.43.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.43.0 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.43.0 // indirect
	go.opentelemetry.io/contrib/propagators/ot v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.1-0.20260521080857-e5bdc311108b // indirect
	go.opentelemetry.io/otel/metric/x v0.0.0-20260521080857-e5bdc311108b // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260519071638-aa98bba5eb94 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260519071638-aa98bba5eb94 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/propagators/b3 => ../propagators/b3

replace go.opentelemetry.io/contrib => ..

replace go.opentelemetry.io/contrib/propagators/aws => ../propagators/aws

replace go.opentelemetry.io/contrib/propagators/autoprop => ../propagators/autoprop

replace go.opentelemetry.io/contrib/propagators/ot => ../propagators/ot

replace go.opentelemetry.io/contrib/propagators/jaeger => ../propagators/jaeger

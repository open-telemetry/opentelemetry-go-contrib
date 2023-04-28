module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin/test

go 1.19

require (
	github.com/gin-gonic/gin v1.8.2
	github.com/stretchr/testify v1.8.2
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.41.0
	go.opentelemetry.io/otel v1.15.0
	go.opentelemetry.io/otel/sdk v1.15.0
	go.opentelemetry.io/otel/trace v1.15.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.11.1 // indirect
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/ugorji/go/codec v1.2.9 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin => ../

replace go.opentelemetry.io/contrib/propagators/b3 => ../../../../../../propagators/b3

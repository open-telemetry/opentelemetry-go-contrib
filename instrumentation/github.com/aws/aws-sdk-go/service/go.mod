module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service

go 1.15

require (
	github.com/aws/aws-sdk-go v1.35.3
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib v0.13.0
	go.opentelemetry.io/otel v0.13.0
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace (
	go.opentelemetry.io/contrib => ../../../../..
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => ../../../../net/http/otelhttp
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

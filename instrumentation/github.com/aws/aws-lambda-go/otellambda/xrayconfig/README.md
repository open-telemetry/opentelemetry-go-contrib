# Recommended Configurations for OpenTelemetry AWS Lambda Instrumentation with AWS X-Ray

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module provides recommended configuration options for [`AWS Lambda Instrumentation`](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/aws/aws-lambda-go/otellambda) when using [AWS X-Ray](https://aws.amazon.com/xray/). By using this configuration, trace context will automatically be extracted from incoming requests with the `X-Amzn-Trace-Id` header if present. Trace context will also always be injected using the `X-Amzn-Trace-Id` format into downstream requests from the Lambda function.

## Installation

```bash
go get -u go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig
```

## Usage

Create a sample Lambda Go application instrumented by the `otellambda` package such as below.

```go
package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
)

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
	return fmt.Sprintf("Hello %s!", name.Name ), nil
}

func main() {
	lambda.Start(otellambda.InstrumentHandler(HandleRequest))
}
```

Now configure the instrumentation with the provided options to export traces to AWS X-Ray via [the OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) running as a Lambda Extension. Instructions for running the OTel Collector as a Lambda Extension can be found in the [AWS OpenTelemetry Documentation](https://aws-otel.github.io/docs/getting-started/lambda).

```go
// Add import
import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"

// add options to InstrumentHandler call
func main() {
	lambda.Start(otellambda.InstrumentHandler(HandleRequest, xrayconfig.WithRecommendedOptions()...))
}
```
## Recommended AWS Lambda Instrumentation Options

| Instrumentation Option | Recommended Value | Exported As |
| --- | --- | --- |
| `WithTracerProvider` | An `sdktrace.TracerProvider` configured to export in batches to an OTel Collector running locally in Lambda | Not individually exported. Can only be used via `WithRecommendedOptions()`
| `WithFlusher` | An `otellambda.Flusher` which yields before calling ForceFlush on the configured `sdktrace.TracerProvider`. Yielding mitigates data delays caused by asynchronous nature of batching TracerProvider when in Lambda | Not individually exported. Can only be used via `WithRecommendedOptions()`
| `WithEventToCarrier` | Function which reads X-Ray TraceID from Lambda environment and inserts it into a `propagtation.TextMapCarrier` | Individually exported as `WithEventToCarrier()`, also included in `WithRecommendedOptions()`
| `WithPropagator` | An `xray.propagator` | Individually exported as `WithPropagator()`, also included in `WithRecommendedOptions()`


## Useful links

- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For more about OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]


## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions

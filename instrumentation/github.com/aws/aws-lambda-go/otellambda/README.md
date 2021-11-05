# OpenTelemetry AWS Lambda Instrumentation for Golang

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module provides instrumentation for [`AWS Lambda`](https://docs.aws.amazon.com/lambda/latest/dg/golang-handler.html).

## Installation

```bash
go get -u go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda
```

## example
See [./example](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/aws/aws-lambda-go/otellambda/example)

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

func HandleRequest(ctx context.Context) (error) {
	fmt.Println("Hello World!" )
	return nil
}

func main() {
	ctx := context.Background()
	lambda.Start(otellambda.InstrumentHandler(HandleRequest(ctx)))
}
```

Now configure the instrumentation with the provided options to export traces to AWS X-Ray via [the OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) running as a Lambda Extension. Instructions for running the OTel Collector as a Lambda Extension can be found in the [AWS OpenTelemetry Documentation](https://aws-otel.github.io/docs/getting-started/lambda).

```go
// Add imports
import (
    "context"
    "fmt"

    "github.com/aws/aws-lambda-go/lambda"
    "go.opentelemetry.io/contrib/propagators/aws/xray"
    "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
    "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
    "go.opentelemetry.io/otel"
)

// add options to WrapHandlerFunction call
func main() {
    ctx := context.Background()

    tp, err := xrayconfig.PrepareTracerProvider(ctx)
    if err != nil {
        fmt.Printf("error creating tracer provider: %v", err)
    }

    defer func(ctx context.Context) {
        err := tp.Shutdown(ctx)
        if err != nil {
            fmt.Printf("error shutting down tracer provider: %v", err)
        }
    }(ctx)

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(xray.Propagator{})
    
    lambda.Start(otellambda.InstrumentHandler(HandleRequest(ctx), xrayconfig.AllRecommendedOptions(tp)...))
}
```

## AWS Lambda Instrumentation Options

| Options | Input Type  | Description | Default |
| --- | --- | --- | --- |
| `WithTracerProvider` | `trace.TracerProvider` | Provide a custom `TracerProvider` for creating spans. Consider using the [AWS Lambda Resource Detector][lambda-detector-url] with your tracer provider to improve tracing information. | `otel.GetTracerProvider()`
| `WithFlusher` | `otellambda.Flusher`  | This instrumentation will call the `ForceFlush` method of its `Flusher` at the end of each invocation. Should you be using asynchronous logic (such as `sddktrace's BatchSpanProcessor`) it is very import for spans to be `ForceFlush`'ed before [Lambda freezes](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-context.html) to avoid data delays. | `Flusher` with noop `ForceFlush`
| `WithEventToCarrier` | `func(eventJSON []byte) propagation.TextMapCarrier{}` | Function for providing custom logic to support retrieving trace header from different event types that are handled by AWS Lambda (e.g., SQS, CloudWatch, Kinesis, API Gateway) and returning them in a `propagation.TextMapCarrier` which a Propagator can use to extract the trace header into the context. | Function which returns an empty `TextMapCarrier` - new spans will be part of a new Trace and have no parent past Lambda instrumentation span
| `WithPropagator` | `propagation.Propagator` | The `Propagator` the instrumentation will use to extract trace information into the context. | `otel.GetTextMapPropagator()` |

## Useful links

- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For more about OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url] 

## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions
[lambda-detector-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/detectors/aws/lambda

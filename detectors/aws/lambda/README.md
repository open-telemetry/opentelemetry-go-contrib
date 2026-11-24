# OpenTelemetry AWS Lambda Resource Detector for Golang

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module detects resource attributes available in AWS Lambda.

## Installation

```bash
go get -u go.opentelemetry.io/contrib/detectors/aws/lambda
```

## Usage

Create a sample Lambda Go application such as below.

```go
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	sdktrace "go.opencensus.io/otel/sdk/trace"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
)

func main() {
	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(context.Background())
	if err != nil {
		fmt.Printf("failed to detect lambda resources: %v\n", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
	)
	lambda.Start(<some lambda handler>)
}
```

Now your `TracerProvider` will have the following resource attributes and attach them to new spans:

| Resource Attribute | Example Value |
| --- | --- |
| `cloud.provider` | aws
| `cloud.platform` | aws_lambda
| `cloud.region` | us-east-1
| `faas.name` | MyLambdaFunction
| `faas.version` | $LATEST
| `faas.instance` | 2021/06/28/[$LATEST]2f399eb14537447da05ab2a2e39309de
| `faas.max_memory` | 134217728
| `aws.log.group.names` | ["/aws/lambda/MyLambdaFunction"]
| `aws.log.stream.names` | ["2021/06/28/[$LATEST]2f399eb14537447da05ab2a2e39309de"]

`faas.max_memory` is reported in bytes, as the semantic conventions require. The
`AWS_LAMBDA_FUNCTION_MEMORY_SIZE` environment variable reports MiB, so a 128 MiB function reports
`134217728`.

Every attribute other than `cloud.provider`, `cloud.platform` and `faas.name` is only emitted when its environment variable is set.

To emit a subset of the attributes, pass a filter:

```go
detector := lambdadetector.NewResourceDetector(
	lambdadetector.WithAttributeFilter(func(kv attribute.KeyValue) bool {
		return kv.Key != semconv.AWSLogStreamNamesKey
	}),
)
```

Of note, `faas.id` and `cloud.account.id` are not set by the Lambda resource detector because they are not available outside a Lambda invocation. For this reason, when using the AWS Lambda Instrumentation these attributes are set as additional span attributes.

## Useful links

- For more on FaaS attribute conventions, visit <https://opentelemetry.io/docs/specs/semconv/faas/faas-spans/>
- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For more about OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]

## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/lambda.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/lambda
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions

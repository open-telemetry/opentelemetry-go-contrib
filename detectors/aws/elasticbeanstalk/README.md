# OpenTelemetry AWS Elastic Beanstalk Resource Detector for Golang

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module detects resource attributes available in AWS Elastic Beanstalk by reading
the AWS X-Ray configuration file. [X-Ray must be enabled][xray-url] on the Beanstalk
environment for the configuration file to be present.

## Installation

```bash
go get -u go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk
```

## Usage

```go
package main

import (
	"context"

	ebdetector "go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk"
	"go.opentelemetry.io/otel/sdk/resource"
)

func main() {
	res, err := resource.New(context.Background(),
		resource.WithDetectors(ebdetector.NewResourceDetector()),
	)
}
```

## Detected Attributes

| Resource Attribute | Example Value |
| --- | --- |
| `cloud.provider` | aws |
| `cloud.platform` | aws_elastic_beanstalk |
| `deployment.environment.name` | production |
| `deployment.id` | 23 |
| `service.version` | env-version-1234 |

## Useful links

- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For semantic conventions on cloud attributes, visit: <https://opentelemetry.io/docs/specs/semconv/resource/cloud/>
- For more about OpenTelemetry Go SDK: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]

## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions
[xray-url]: https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/environment-configuration-debugging.html

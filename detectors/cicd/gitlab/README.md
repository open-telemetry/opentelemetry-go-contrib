# OpenTelemetry Gitlab CI/CD Detector for Golang

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module detects resource attributes available in Gitlab CI Pipeline.

## Installation

```bash
go get -u go.opentelemetry.io/contrib/detectors/cicd/gitlab
```

## Usage

Create a sample Go application such as below.

```go
package main

import (
	sdktrace "go.opencensus.io/otel/sdk/trace"
	gitlabdetector "go.opentelemetry.io/contrib/detectors/cicd/gitlab"
)

func main() {
	detector := gitlabdetector.NewResourceDetector()
	res, err := detector.Detect(context.Background())
	if err != nil {
		fmt.Printf("failed to detect gitlab CICD resources: %v\n", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
	)

	...
}
```

Now your `TracerProvider` will have the following resource attributes and attach them to new spans:

| Resource Attribute              | Example Value                  |
|---------------------------------|--------------------------------|
| cicd.pipeline.name              | test                           |
| cicd.pipeline.task.run.id       | 123                            |
| cicd.pipeline.task.name         | unit-test                      |
| cicd.pipeline.task.type         | test                           |
| cicd.pipeline.run.id            | 12345                          |
| cicd.pipeline.task.run.url.full | https://gitlab/job/123         |
| vcs.repository.ref.name         | myProject                      |
| vcs.repository.ref.type         | branch                         |
| vcs.repository.change.id        | 12                             |
| vcs.repository.url.full         | https://gitlab/myOrg/myProject |

## Useful links

- For more on CI/CD pipeline attribute conventions,
  visit <https://opentelemetry.io/docs/specs/semconv/attributes-registry/cicd/>
- For more on VCS attribute conventions, visit <https://opentelemetry.io/docs/specs/semconv/attributes-registry/vcs/>
- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For more about OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]

## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE

[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat

[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/cicd/gitlab.svg

[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/cicd/gitlab

[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions

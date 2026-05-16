# OpenTelemetry Docker Resource Detector for Go

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module detects resource attributes for processes running inside Docker containers.
It uses the [Moby Docker client](https://github.com/moby/moby) to query the Docker daemon
for container and host information.

The Docker daemon socket must be accessible (e.g. `/var/run/docker.sock` mounted into the container).

## Installation

```bash
go get -u go.opentelemetry.io/contrib/detectors/docker
```

## Usage

```go
package main

import (
    "context"

    dockerdetector "go.opentelemetry.io/contrib/detectors/docker"
    "go.opentelemetry.io/otel/sdk/resource"
)

func main() {
    res, err := resource.New(context.Background(),
        resource.WithDetectors(dockerdetector.NewResourceDetector()),
    )
}
```

## Detected Attributes

| Resource Attribute       | Example Value    | Source                                  |
| ------------------------ | ---------------- | --------------------------------------- |
| `host.name`              | `my-docker-host` | Docker host machine name                |
| `os.type`                | `linux`          | Docker host operating system type       |
| `container.name`         | `/my-container`  | Container name as registered with daemon|
| `container.image.name`   | `golang:1.25`    | Image the container was started from    |

## Useful links

- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For semantic conventions on container attributes, visit: <https://opentelemetry.io/docs/specs/semconv/resource/container/>
- For more about OpenTelemetry Go SDK: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]

## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/docker.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/docker
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions

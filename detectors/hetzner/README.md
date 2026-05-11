# Hetzner Cloud Resource Detector for Go

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module detects resource attributes available on
[Hetzner Cloud](https://www.hetzner.com/cloud) servers by querying the Hetzner
Cloud Instance Metadata Service.

## Installation

```bash
go get go.opentelemetry.io/contrib/detectors/hetzner
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/contrib/detectors/hetzner"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithDetectors(hetzner.NewResourceDetector()),
	)
	if err != nil {
		log.Fatalf("failed to detect Hetzner resources: %v", err)
	}
	fmt.Println(res.String())

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp
}
```

## Detected Resource Attributes

The following attributes are set when the process is running on a Hetzner Cloud server:

| Resource Attribute        | Example Value          |
|---------------------------|------------------------|
| `cloud.provider`          | `hetzner_cloud`        |
| `cloud.platform`          | `hetzner_cloud_hcloud` |
| `cloud.region`            | `nbg1`                 |
| `cloud.availability_zone` | `nbg1-dc3`             |
| `host.id`                 | `987654321`            |
| `host.name`               | `srv-123`              |

When the process is not running on a Hetzner Cloud server, no attributes are set
and no error is returned.

If the process is running on a Hetzner Cloud server but individual metadata
endpoints are unreachable, the available attributes are returned together with
[`resource.ErrPartialResource`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/resource#ErrPartialResource).

## Useful Links

- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For more about OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
- For the Hetzner Cloud Metadata API: <https://docs.hetzner.cloud/#server-metadata>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]

## License

Apache 2.0 — See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/hetzner.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/hetzner
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions

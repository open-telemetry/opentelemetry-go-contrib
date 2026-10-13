# Docker Resource Detector

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/docker.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/docker-->

This module provides a [`resource.Detector`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/resource#Detector) that detects attributes for processes running inside Docker containers:

- `container.name`
- `container.image.name`
- `container.image.id`
- `container.image.tags`
- `host.name`
- `os.type`

## Usage

```golang
res, err := resource.New(
    context.Background(),
    resource.WithDetectors(docker.NewResourceDetector()),
)
```

The detector queries the Docker daemon via its API and requires the daemon socket to be accessible to the process (e.g. `/var/run/docker.sock` mounted into the container). If the detector cannot confirm the calling process is running inside a Docker container (an unreachable daemon, or a daemon that reports no container matching the process's hostname), an empty resource and no error are returned.

`container.image.name` and `container.image.tags` are omitted when the container was referenced by a bare image ID (e.g. `docker run sha256:<id>`), since there is then no name or tag to report; `container.image.id` still identifies the image in that case.

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

The detector queries the Docker daemon via its API and requires the daemon socket to be accessible to the process (e.g. `/var/run/docker.sock` mounted into the container). The detector attempts to identify the container using its ID from cgroup data, with `/proc/self/mountinfo` as a fallback for private cgroup namespaces. If neither source provides an ID, or the Docker daemon does not recognize the ID, the detector falls back to the process hostname. If the detector cannot confirm that the calling process is running inside a Docker container (e.g. the daemon is unreachable or no matching container is found), an empty resource and no error are returned.

`container.image.name` and `container.image.tags` are omitted when the container was referenced by a bare image ID (e.g. `docker run sha256:<id>`), since there is then no name or tag to report; `container.image.id` still identifies the image in that case.

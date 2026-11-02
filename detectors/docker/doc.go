// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package docker provides a [resource.Detector] which supports detecting
attributes for processes running inside Docker containers.

The detector queries the Docker daemon via its API and requires the daemon
socket to be accessible to the process (e.g. /var/run/docker.sock mounted
into the container). If [resource.Detector.Detect] cannot confirm the calling
process is running inside a Docker container, it returns an empty resource
and no error; this covers both an unreachable daemon and a daemon that is
reachable but reports no container matching the process's hostname. If the
container is identified but some attributes cannot be retrieved, a partial
resource is returned together with [resource.ErrPartialResource].

According to semantic conventions for [container], [host], and [os] attributes,
each of the following attributes is detected:

  - container.name
  - container.image.name
  - container.image.id
  - container.image.tags
  - host.name
  - os.type

container.image.name and container.image.tags are omitted when the
container was referenced by a bare image ID (e.g. "docker run
sha256:<id>"), since there is then no name or tag to report;
container.image.id still identifies the image in that case.

[container]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/container.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[os]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/os.md
*/
package docker

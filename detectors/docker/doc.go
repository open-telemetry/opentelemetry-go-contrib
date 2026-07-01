// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package docker provides a [resource.Detector] which supports detecting
attributes for processes running inside Docker containers.

The detector queries the Docker daemon via its API and requires the daemon
socket to be accessible to the process (e.g. /var/run/docker.sock mounted
into the container). If the socket is unavailable or any daemon call fails,
[resource.Detector.Detect] returns an empty resource with error.

According to semantic conventions for [container], [host], and [os] attributes,
each of the following attributes is detected:

  - container.name
  - container.image.name
  - host.name
  - os.type

[container]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/container.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[os]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/os.md
*/
package docker // import "go.opentelemetry.io/contrib/detectors/docker"

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package system provides a [resource.Detector] that detects host- and OS-level
resource attributes from the local system.

The following attributes are detected when available:

  - host.name (configurable via [WithHostnameSources])
  - host.id
  - host.arch
  - host.ip
  - host.mac
  - host.cpu.vendor.id
  - host.cpu.family
  - host.cpu.model.id
  - host.cpu.model.name
  - host.cpu.stepping
  - host.cpu.cache.l2.size
  - os.type
  - os.description

[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[os]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/os.md
*/
package system // import "go.opentelemetry.io/contrib/detectors/system"

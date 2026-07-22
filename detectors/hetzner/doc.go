// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package hetzner provides a [resource.Detector] which supports detecting
attributes specific to Hetzner Cloud servers.

According to semantic conventions for [cloud] and [host] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.availability_zone
  - host.id
  - host.name

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package hetzner

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package cvm provides a [resource.Detector] which supports detecting attributes
specific to Tencent Cloud CVM instances.

According to semantic conventions for [cloud] and [host] attributes, each of
the following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.region
  - cloud.availability_zone
  - host.id
  - host.name
  - host.image.id
  - host.type

The cloud.account.id value is the Tencent Cloud AppID reported by the metadata
service, not the account UIN.

The metadata service serves each value as a separate plain text document, so a
detection performs one request per attribute. Values are trimmed of surrounding
whitespace before being reported. Detection is all-or-nothing: if any of those
requests fails, no instance attributes are reported at all, matching the
behavior of the Tencent Cloud CVM detector in the OpenTelemetry Collector.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package cvm

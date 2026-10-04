// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package vultr provides a [resource.Detector] which supports detecting
attributes specific to Vultr Cloud Compute instances.

According to semantic conventions for [cloud] and [host] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - host.id
  - host.name

The cloud.region value is the Vultr region code normalized to lower case: the
metadata value "EWR" is reported as "ewr".

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package vultr

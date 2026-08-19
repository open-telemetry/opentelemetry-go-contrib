// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package scaleway provides a [resource.Detector] which supports detecting
attributes specific to Scaleway Instances.

According to semantic conventions for [cloud] and [host] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.availability_zone
  - cloud.region
  - host.id
  - host.image.id
  - host.image.name
  - host.name
  - host.type

The cloud.region value is derived from the zone reported by the metadata
service by dropping its trailing segment: the zone "nl-ams-1" is reported as the
region "nl-ams".

The metadata service is queried through the Scaleway SDK, which reports a
failure without the HTTP status that caused it. A metadata service that answers
with an error is therefore indistinguishable from one that is not there at all,
and both are reported as not running on Scaleway.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package scaleway

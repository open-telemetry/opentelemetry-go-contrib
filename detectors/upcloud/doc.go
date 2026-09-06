// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package upcloud provides a [resource.Detector] which supports detecting
attributes specific to UpCloud Cloud Servers.

According to semantic conventions for [cloud] and [host] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.region
  - host.id
  - host.name

The cloud.provider value is reported by the metadata service itself, in the
cloud_name field, and is "upcloud" on UpCloud Cloud Servers.

No cloud.platform attribute is emitted: semantic conventions define no platform
value for UpCloud, and the UpCloud detector in opentelemetry-collector-contrib,
which this package is ported from, does not emit one either.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package upcloud

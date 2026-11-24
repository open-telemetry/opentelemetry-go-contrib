// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package classic provides a [resource.Detector] which supports detecting
attributes specific to IBM Cloud Classic Infrastructure (SoftLayer) instances.

According to semantic conventions for [cloud] and [host] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.availability_zone
  - cloud.resource_id
  - host.id
  - host.name

The attributes are read from the SoftLayer Resource Metadata service, which
exposes one endpoint per field. The service is reached over HTTPS at a public
host name and authenticates by source IP; requests are made with proxying
disabled so that instance metadata is never routed through an HTTP(S) proxy.

Every field the service exposes is required. When the process runs on an IBM
Cloud Classic instance but a field cannot be retrieved, its attribute is omitted
and Detect returns the partial resource together with
[resource.ErrPartialResource]. When the metadata service cannot be reached at
all, Detect returns an empty resource and no error.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package classic

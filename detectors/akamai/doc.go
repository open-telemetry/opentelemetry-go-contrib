// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package akamai provides a [resource.Detector] which supports detecting
attributes specific to Akamai Connected Cloud (formerly Linode) compute
instances.

According to semantic conventions for [cloud] and [host] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.region
  - host.id
  - host.name
  - host.type
  - host.image.id
  - host.image.name

The host.id value is the numeric instance ID reported by the metadata service,
not the host_uuid field of the same document.

Detection reads the [Instance Metadata Service] on the link-local address
169.254.169.254. Each detection mints a short lived metadata token and then
fetches the instance document once, for two requests in total. Minting the token
also serves as the availability probe: when the token endpoint is unreachable or
answers with a client error, the process is not running on an Akamai instance
and an empty resource is returned.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[Instance Metadata Service]: https://techdocs.akamai.com/cloud-computing/docs/metadata-service-api
*/
package akamai

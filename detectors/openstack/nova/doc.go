// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package nova provides a [resource.Detector] which supports detecting attributes
specific to OpenStack Nova compute instances.

According to semantic conventions for [cloud] and [host] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.availability_zone
  - host.id
  - host.name
  - host.type

The host.type value is read from the EC2 compatible metadata endpoint, which
not every OpenStack deployment serves. It is therefore detected on a best
effort basis: when it cannot be read the attribute is omitted and no error is
reported.

Instance metadata keys defined by the user or the deployment are not detected
by default. Pass [WithMetaKeyFilter] to emit the selected ones as
openstack.nova.meta.<key> attributes.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package nova

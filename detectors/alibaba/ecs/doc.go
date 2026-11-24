// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package ecs provides a [resource.Detector] which supports detecting attributes
specific to Alibaba Cloud Elastic Compute Service (ECS) instances.

According to semantic conventions for [cloud] and [host] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.region
  - cloud.availability_zone
  - host.id
  - host.name
  - host.image.id
  - host.type

The [instance metadata service] requires a token: the detector first requests
one, then reads each metadata value with that token. The metadata service serves
a single value per path, so one request is made per attribute. When the token
cannot be obtained because nothing answers the metadata address, the process is
not running on an ECS instance and an empty resource is returned. When a token is
obtained but an individual metadata value cannot be read, that attribute is
omitted and a partial resource is returned together with
[resource.ErrPartialResource].

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[instance metadata service]: https://www.alibabacloud.com/help/en/ecs/user-guide/view-instance-metadata
*/
package ecs

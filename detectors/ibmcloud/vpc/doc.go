// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package vpc provides a [resource.Detector] which supports detecting attributes
specific to IBM Cloud VPC virtual server instances.

According to semantic conventions for [cloud] and [host] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.account.id
  - cloud.region
  - cloud.availability_zone
  - cloud.resource_id
  - host.id
  - host.image.id
  - host.image.name
  - host.name
  - host.type

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package vpc // import "go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"

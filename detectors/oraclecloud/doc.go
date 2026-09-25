// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package oraclecloud provides a [resource.Detector] which supports detecting
attributes specific to Oracle Cloud Infrastructure (OCI) instances.

According to semantic conventions for [cloud], [host], and [k8s] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.availability_zone
  - host.id
  - host.name
  - host.type
  - k8s.cluster.name
  - oracle_cloud.realm

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[k8s]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/k8s.md
*/
package oraclecloud

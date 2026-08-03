// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package oraclecloud provides a [resource.Detector] which supports detecting
attributes specific to Oracle Cloud Infrastructure (OCI) instances.

According to semantic conventions for [cloud], [host], [k8s], and
[oracle_cloud] attributes, each of the following attributes is added if it is
available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.availability_zone
  - cloud.resource_id
  - host.id
  - host.name
  - host.type
  - k8s.cluster.name
  - oracle_cloud.realm

The detector queries OCI IMDSv2 at `http://169.254.169.254/opc/v2/instance/`
and uses `hostname` for `host.name`. When the custom metadata key
`oke-cluster-display-name` is present, the detector reports
`cloud.platform=oracle_cloud_oke` and `k8s.cluster.name`.

[cloud]: https://opentelemetry.io/docs/specs/semconv/registry/attributes/cloud/
[host]: https://opentelemetry.io/docs/specs/semconv/registry/attributes/host/
[k8s]: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
[oracle_cloud]: https://opentelemetry.io/docs/specs/semconv/registry/attributes/oracle-cloud/
*/
package oraclecloud

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package alibabacloud provides a [resource.Detector] which supports detecting
attributes specific to Alibaba Cloud ECS instances and to containers running
on Elastic Container Instance (ECI), including pods on Alibaba Cloud
Container Compute Service (ACS).

Both ECS instances and ECI pods expose the same instance metadata service at
100.100.100.200. On an ECI pod the hostname is the pod name and the
instance-id is an "eci-" prefixed identifier. Because semantic conventions
define no cloud platform value for ECI, containers report the same
"alibaba_cloud_ecs" cloud.platform value as ECS instances. On pods scheduled
to regular Alibaba Cloud Container Service for Kubernetes (ACK) nodes the
service answers with the node's metadata, so host.id and host.name identify
the underlying ECS instance.

The metadata service is queried with the hardened IMDSv2 token flow when the
instance supports it, falling back to unauthenticated IMDSv1 requests
otherwise.

The detected attribute set matches the alibaba/ecs detector of the
resourcedetectionprocessor in opentelemetry-collector-contrib. According to
semantic conventions for [cloud] and [host] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.availability_zone
  - cloud.account.id
  - host.id
  - host.name
  - host.type
  - host.image.id

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package alibabacloud

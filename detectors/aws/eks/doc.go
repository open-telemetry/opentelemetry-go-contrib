// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package eks provides a [resource.Detector] which supports detecting
attributes for applications running on AWS Elastic Kubernetes Service (EKS).

According to semantic conventions, each of the following attributes is detected
if available:

  - cloud.provider ("aws")
  - cloud.platform ("aws_eks")
  - cloud.account.id
  - cloud.availability_zone
  - cloud.region
  - host.id
  - host.image.id
  - host.name
  - host.type
  - k8s.cluster.name
  - container.id

The detector can be configured using options such as [WithAttributeFilter],
[WithNodeEnvVar], and [WithAWSConfig].
*/
package eks

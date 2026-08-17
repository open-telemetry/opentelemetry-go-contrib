// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package openshift provides a [resource.Detector] which supports detecting
attributes specific to OpenShift 4 clusters.

The detector reads the cluster-scoped Infrastructure config object from the
OpenShift API server, authenticating with the service account token projected
into the pod. It requires the following RBAC:

  - apiGroups: ["config.openshift.io"]
    resources: ["infrastructures"]
    resourceNames: ["cluster"]
    verbs: ["get"]

According to semantic conventions for [cloud] and [k8s] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - k8s.cluster.name

The k8s.cluster.name value is the infrastructure name of the cluster. Which
cloud attributes are reported depends on the infrastructure the cluster runs
on:

  - AWS, Azure, Google Cloud and IBM Cloud clusters report cloud.provider,
    cloud.platform and cloud.region.
  - OpenStack clusters report only cloud.region. Semantic conventions define no
    cloud.provider value for OpenStack and no cloud.platform value for
    OpenShift on OpenStack.
  - Clusters that do not run on a cloud provider, such as bare metal, vSphere
    and oVirt, report no cloud attributes at all. This is not treated as a
    partial resource.

Region values are normalized to lower case.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[k8s]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/k8s.md
*/
package openshift

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package azureaks provides a [resource.Detector] which supports detecting
attributes specific to Azure Kubernetes Service (AKS).

According to semantic conventions for [cloud] and [k8s] attributes, each of the
following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - k8s.cluster.name

Detection requires both that the KUBERNETES_SERVICE_HOST environment variable is
set, which the kubelet does for every pod, and that the Azure Instance Metadata
Service is reachable. When either is not the case an empty resource is returned.

The k8s.cluster.name value is parsed from the infrastructure resource group name
reported by the Azure Instance Metadata Service. AKS generates that group as
MC_<resource group>_<cluster name>_<location>, from which the cluster name is
extracted. When a custom infrastructure resource group is used, or when the
resource group or cluster name itself contains an underscore, the resource group
name is reported unchanged: Azure does not allow two AKS clusters to share an
infrastructure resource group name, so it remains a unique cluster identifier.

Use [WithAttributeFilter] to exclude attributes, for example
attribute.NewDenyKeysFilter(semconv.K8SClusterNameKey) to leave k8s.cluster.name
out of the resource.

AKS nodes are also Azure virtual machines. Combining this detector with the
[go.opentelemetry.io/contrib/detectors/azure/azurevm] detector will produce
conflicting cloud.platform values; use one or the other.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[k8s]: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
*/
package azureaks

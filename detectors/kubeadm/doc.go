// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package kubeadm provides a [resource.Detector] which supports detecting
attributes of the [kubeadm]-provisioned Kubernetes cluster the process is
running in.

According to semantic conventions for [k8s] attributes, each of the following
attributes is added if it is available:

  - k8s.cluster.name
  - k8s.cluster.uid

The cluster name is read from the clusterName field of the ClusterConfiguration
document kubeadm stores in the kubeadm-config ConfigMap of the kube-system
namespace. The cluster UID is derived from the UID of the kube-system namespace
itself.

Detection requires the following RBAC:

	rules:
	  - apiGroups: [""]
	    resources: ["configmaps"]
	    resourceNames: ["kubeadm-config"]
	    verbs: ["get"]
	  - apiGroups: [""]
	    resources: ["namespaces"]
	    resourceNames: ["kube-system"]
	    verbs: ["get"]

The detector returns an empty resource and no error when the process is not
running inside a Kubernetes cluster, and when it is running inside a cluster
that was not provisioned with kubeadm. A cluster is considered not to be a
kubeadm cluster when the kubeadm-config ConfigMap does not exist.

The k8s.cluster.uid attribute is derived from the same source as the attribute
of the same name in [go.opentelemetry.io/contrib/detectors/k8sapi]. Using both
detectors together is supported: they report an identical value, so merging
their resources does not conflict.

[kubeadm]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/
[k8s]: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
*/
package kubeadm

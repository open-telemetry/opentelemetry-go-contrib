// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package k8sapi provides a [resource.Detector] which supports detecting
attributes from the Kubernetes API.

According to semantic conventions for [k8s] attributes,
each of the following attributes is added if it is available:

  - k8s.node.name
  - k8s.node.uid
  - k8s.cluster.uid

Node attributes require the K8S_NODE_NAME environment variable to be set,
typically via the Kubernetes downward API:

	env:
	  - name: K8S_NODE_NAME
	    valueFrom:
	      fieldRef:
	        fieldPath: spec.nodeName

Node attributes (k8s.node.name, k8s.node.uid) require the following RBAC:

  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get"]

The cluster UID is derived from the kube-system namespace UID and requires
the following RBAC:

  - apiGroups: [""]
    resources: ["namespaces"]
    resourceNames: ["kube-system"]
    verbs: ["get"]

[k8s]: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
*/
package k8sapi // import "go.opentelemetry.io/contrib/detectors/k8sapi"

# Kubernetes API Resource Detector

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/k8sapi.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/k8sapi-->

This module provides a [`resource.Detector`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/resource#Detector) that detects [k8s semantic convention](https://opentelemetry.io/docs/specs/semconv/resource/k8s/) attributes using the Kubernetes API:

- `k8s.node.name`
- `k8s.node.uid`
- `k8s.cluster.uid`

## Usage

```golang
res, err := resource.New(
    context.Background(),
    resource.WithDetectors(k8sapi.NewResourceDetector()),
)
```

Node attributes require the `K8S_NODE_NAME` environment variable to be set, typically via the Kubernetes downward API:

```yaml
env:
  - name: K8S_NODE_NAME
    valueFrom:
      fieldRef:
        fieldPath: spec.nodeName
```

Node attributes (`k8s.node.name`, `k8s.node.uid`) require the following RBAC:

```yaml
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get"]
```

The cluster UID is derived from the `kube-system` namespace UID and requires the following RBAC:

```yaml
- apiGroups: [""]
  resources: ["namespaces"]
  resourceNames: ["kube-system"]
  verbs: ["get"]
```

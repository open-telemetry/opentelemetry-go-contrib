# AWS EKS Resource Detector

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/eks.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/eks-->

This module provides a [`resource.Detector`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/resource#Detector) that detects attributes for applications running on Amazon Elastic Kubernetes Service (EKS):

- `cloud.provider`
- `cloud.platform`
- `k8s.cluster.name`
- `container.id`

## Usage

```go
import (
    "context"

    "go.opentelemetry.io/contrib/detectors/aws/eks"
    "go.opentelemetry.io/otel/sdk/resource"
)

res, err := resource.New(
    context.Background(),
    resource.WithDetectors(eks.NewResourceDetector()),
)
```

The detector identifies an EKS environment by verifying the presence of IAM Roles for Service Accounts (IRSA) tokens, EKS Pod Identity authorization tokens, EKS cluster version identifiers (`-eks-`), and the `kube-system/aws-auth` ConfigMap.

# AWS EKS Resource Detector

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/eks.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/eks-->

This module provides a [`resource.Detector`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/resource#Detector) that detects attributes for applications running on Amazon Elastic Kubernetes Service (EKS):

- `cloud.provider`
- `cloud.platform`
- `cloud.account.id`
- `cloud.availability_zone`
- `cloud.region`
- `host.id`
- `host.image.id`
- `host.name`
- `host.type`
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

Host and node attributes are detected via the EC2 Instance Metadata Service (IMDS). In environments where IMDS access is disabled or restricted (such as EKS Auto Mode or IMDS hop limit set to 1), the detector falls back to reading the node provider ID via the Kubernetes API using the node name from an environment variable (configured with `WithNodeEnvVar`, defaulting to `K8S_NODE_NAME`).

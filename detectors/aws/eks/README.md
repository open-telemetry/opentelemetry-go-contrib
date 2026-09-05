# OpenTelemetry AWS EKS Resource Detector for Golang

[![Go Reference][goref-image]][goref-url]
[![Apache License][license-image]][license-url]

This module detects resource attributes available to workloads running on
[Amazon EKS](https://aws.amazon.com/eks/).

## Installation

```bash
go get -u go.opentelemetry.io/contrib/detectors/aws/eks
```

## Usage

```go
// Instantiate a new EKS resource detector.
eksResourceDetector := eks.NewResourceDetector()
res, err := eksResourceDetector.Detect(context.Background())
if err != nil {
	fmt.Printf("failed to detect EKS resources: %v\n", err)
}

tp := sdktrace.NewTracerProvider(
	sdktrace.WithResource(res),
)
```

The detector only talks to the in-cluster Kubernetes API server and the local
filesystem. It does not call any AWS API, and therefore needs no AWS
credentials.

## Detected attributes

| Resource attribute | Example value | Source |
| --- | --- | --- |
| `cloud.provider` | `aws` | Constant, set once the environment is confirmed to be EKS. |
| `cloud.platform` | `aws_eks` | Constant, set once the environment is confirmed to be EKS. |
| `k8s.cluster.name` | `my-cluster` | Key `cluster.name` of the `amazon-cloudwatch/cluster-info` ConfigMap. |
| `container.id` | `a3b1...` (64 hex chars) | Last 64 characters of a `/docker/<id>` line in `/proc/self/cgroup`. |

`k8s.cluster.name` is omitted when the ConfigMap has no `cluster.name` key. That
ConfigMap is created by the CloudWatch Container Insights agent rather than by
EKS itself, so clusters without Container Insights installed will not report a
cluster name.

## Detection and failure behavior

`Detect` reports EKS when both of the following hold:

1. `/var/run/secrets/kubernetes.io/serviceaccount/token` and `ca.crt` exist,
   which confirms the process runs inside *some* Kubernetes cluster.
2. The `kube-system/aws-auth` ConfigMap can be read through the in-cluster API
   server. This ConfigMap is provisioned by EKS, so reading it distinguishes EKS
   from vanilla Kubernetes. The pod's service account needs `get` permission on
   it.

| Condition | `Detect` returns |
| --- | --- |
| Not running in Kubernetes (`rest.ErrNotInCluster`) | empty resource, no error |
| Any other client construction failure | `nil`, error |
| Service account token or certificate missing | empty resource, no error |
| `kube-system/aws-auth` unreadable | `nil`, error |
| `kube-system/aws-auth` absent | empty resource, no error |
| `amazon-cloudwatch/cluster-info` unreadable | `nil`, error |
| `amazon-cloudwatch/cluster-info` has no `cluster.name` | resource without `k8s.cluster.name`, no error |
| `/proc/self/cgroup` unreadable, or no Docker-style cgroup line | `nil`, error |

## Differences from the Collector's `resourcedetection` processor

The [`resourcedetectionprocessor`][rdp-url] ships its own EKS detector. The two
implementations have diverged; reconciling them is tracked in
[open-telemetry/opentelemetry-go-contrib#8944][issue-url]. Neither is currently a
superset of the other.

| Concern | This detector | `resourcedetectionprocessor` |
| --- | --- | --- |
| EKS detection | Service account files present **and** `kube-system/aws-auth` readable | `KUBERNETES_SERVICE_HOST` set, then any of: IRSA token path containing `eks.amazonaws.com`, Pod Identity token path containing `eks-pod-identity`, OIDC issuer containing `oidc.eks.`, cluster version containing `-eks-` |
| `k8s.cluster.name` source | `amazon-cloudwatch/cluster-info` ConfigMap | EC2 instance tags `aws:eks:cluster-name`, `eks:cluster-name`, or the `kubernetes.io/cluster/` prefix, via `ec2:DescribeInstances` |
| Node attributes | Not collected | `host.id`, `host.name`, `host.type`, `host.image.id`, `cloud.region`, `cloud.availability_zone`, `cloud.account.id`, all disabled by default |
| `container.id` | Collected | Not collected |
| Partial results | A failure after EKS is confirmed discards all attributes | Sub-step failures are logged and a partial resource is emitted, unless `fail_on_missing_metadata` is set |
| AWS credentials | Not required | Required for the EC2 API calls |
| Semantic conventions | v1.43.0 | v1.40.0 |

## Useful links

- For more on Kubernetes attribute conventions, visit <https://opentelemetry.io/docs/specs/semconv/resource/k8s/>
- For more information on OpenTelemetry, visit: <https://opentelemetry.io/>
- For more about OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
- For help or feedback on this project, join us in [GitHub Discussions][discussions-url]

## License

Apache 2.0 - See [LICENSE][license-url] for more information.

[license-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/eks.svg
[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/eks
[discussions-url]: https://github.com/open-telemetry/opentelemetry-go/discussions
[rdp-url]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor
[issue-url]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/8944

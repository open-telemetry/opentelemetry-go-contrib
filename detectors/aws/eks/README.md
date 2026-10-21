# AWS EKS Resource Detector

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/eks.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/eks)

This module detects OpenTelemetry resource attributes for workloads running on
Amazon EKS. `NewResourceDetector` reports the EKS platform, cluster name, and
container ID available from the Kubernetes environment.

`NewResourceDetectorWithOptions(WithClusterARN())` also attempts to report
`aws.eks.cluster.arn`. The option calls EKS `DescribeCluster` and accepts the ARN
only when the returned OIDC issuer matches the issuer published by the live
Kubernetes API server. See the [AWS detector documentation](../README.md#eks)
for permissions, prerequisites, timeout, and error behavior.

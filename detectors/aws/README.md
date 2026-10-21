# AWS Resource Detectors

## EC2

Sample code snippet to initialize EC2 resource detector

```
// Instantiate a new EC2 Resource detector
ec2ResourceDetector := ec2.NewResourceDetector()
resource, err := ec2ResourceDetector.Detect(context.Background())
```

EC2 resource detector captures following EC2 instance environment attributes

```
cloud.region
cloud.availability_zone
cloud.account.id
host.id
host.image.id
host.type
```

## ECS

Sample code snippet to initialize ECS resource detector

```
// Instantiate a new ECS Resource detector
ecsResourceDetector := ecs.NewResourceDetector()
resource, err := ecsResourceDetector.Detect(context.Background())
```

ECS resource detector captures following ECS environment attributes

```
cloud.region
cloud.availability_zone
cloud.account.id
cloud.resource_id
container.name
container.id
aws.ecs.cluster.arn
aws.ecs.container.arn
aws.ecs.launchtype
aws.ecs.task.arn
aws.ecs.task.family
aws.ecs.task.revision
aws.log.group.arns
aws.log.group.names
aws.log.stream.arns
aws.log.stream.names
```

## EKS

Sample code snippet to initialize EKS resource detector

```
// Instantiate a new EKS Resource detector
eksResourceDetector := eks.NewResourceDetector()
resource, err := eksResourceDetector.Detect(context.Background())
```

EKS resource detector captures the following attributes by default:

```
k8s.cluster.name
container.id
```

To also detect `aws.eks.cluster.arn`, enable the opt-in ARN lookup:

```go
eksResourceDetector := eks.NewResourceDetectorWithOptions(eks.WithClusterARN())
resource, err := eksResourceDetector.Detect(context.Background())
```

The lookup calls `eks:DescribeCluster` and reports the ARN only when the OIDC
issuer returned by EKS matches the issuer published by the live Kubernetes API
server. It requires all of the following:

- the `amazon-cloudwatch/cluster-info` ConfigMap contains `cluster.name`;
- the AWS SDK can resolve credentials and a Region;
- those credentials allow `eks:DescribeCluster` for that cluster; and
- the Kubernetes service account can read
  `/.well-known/openid-configuration`, for example through Kubernetes' built-in
  `system:service-account-issuer-discovery` ClusterRole.

The OIDC request is bounded to 10 seconds. AWS configuration and
`DescribeCluster` share a separate 10-second bound, so enabling the option adds
at most 20 seconds when no shorter caller deadline exists. If the ARN cannot be
established, the detector returns the normal EKS attributes together with
`resource.ErrPartialResource`; callers that treat any non-nil detector error as
fatal should account for that before enabling the option. `k8s.cluster.name`
remains present on an issuer mismatch because its source is the in-cluster
`cluster-info` ConfigMap. A mismatch can mean that value is stale, but it can
also mean the AWS credentials point at a different account or Region; the ARN
check cannot distinguish those cases, so it does not change the existing
cluster-name behavior. The default `NewResourceDetector()` behavior makes
neither of these additional requests. Each opted-in process calls
`DescribeCluster` during detection, so large deployments should account for the
shared EKS API quota. Throttled calls return the normal attributes with
`resource.ErrPartialResource` and no cluster ARN.

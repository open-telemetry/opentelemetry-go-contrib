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

The lookup calls `eks:DescribeCluster` and reports the ARN only when the
returned OIDC issuer matches the issuer published by the live Kubernetes API
server. This prevents a stale or copied cluster name from producing an ARN for
a different cluster.

The option requires `cluster.name` in the `amazon-cloudwatch/cluster-info`
ConfigMap, AWS credentials and the EKS cluster's Region, `eks:DescribeCluster`
permission, and access to `/.well-known/openid-configuration`. Kubernetes'
built-in `system:service-account-issuer-discovery` ClusterRole provides the
last permission.

The OIDC request is bounded to 10 seconds. AWS configuration and
`DescribeCluster` share another 10-second bound. A shorter caller deadline
limits the whole call, including concurrent client initialization. If identity
cannot be established, the detector returns the normal EKS attributes with
`resource.ErrPartialResource` and no ARN. The default constructor makes no
additional requests. This module now depends on the AWS SDK; enabling the option
controls the additional requests and performs one `DescribeCluster` call per
detection.

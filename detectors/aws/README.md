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

EKS resource detector captures following EKS environment attributes
```
k8s.cluster.name
container.id
```

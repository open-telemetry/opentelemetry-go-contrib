# AWS EC2 Resource Detector

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/aws/ec2/v2.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/aws/ec2/v2)

The detector reads EC2 instance metadata and returns the corresponding OpenTelemetry resource attributes.

Customize the AWS SDK retry limits with detector options:

```go
detector := ec2.NewResourceDetectorWithOptions(
	ec2.WithMaxAttempts(10),
	ec2.WithMaxBackoff(5*time.Minute),
)
```

Both options are optional. Non-positive values cause resource detection to return an error.

Add EC2 instance tags as `ec2.tag.<key>` resource attributes by reading them from the instance metadata service:

```go
detector := ec2.NewResourceDetectorWithOptions(
	ec2.WithIMDSTags(regexp.MustCompile(`^Name$`), regexp.MustCompile(`^team-`)),
)
```

Only tags whose keys match at least one pattern are added.
Access to tags in instance metadata must be enabled for the instance; otherwise no tag attributes are added.

# Service Resource detector

The service resource detector supports detecting attributes specific services.

## Usage

```golang
// Instantiate a new host resource detector
serviceResourceDetector := service.New()
resource, err := serviceResourceDetector.Detect(context.Background())
```

## Supported attributes

According to semantic conventions for
[service](https://github.com/open-telemetry/semantic-conventions/tree/main/docs/resource#service-experimental)
attributes, the following attributes is added:

* `service.instance.id`

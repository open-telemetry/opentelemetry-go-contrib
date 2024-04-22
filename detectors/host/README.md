# Host Resource detector

The host resource detector supports detecting host-specific attributes on physical hosts.

## Usage

```golang
// Instantiate a new host resource detector
hostResourceDetector := host.NewResourceDetector()
resource, err := hostResourceDetector.Detect(context.Background())
```

To populate optional attributes, the resource detector constructor accepts functional options `WithIPAddresses` to enable `host.ip`, and `WithMACAddresses` to enable `host.mac`.

```golang
// Instantiate a new host resource detector with all opt-in attributes
hostResourceDetector := host.NewResourceDetector(
	WithIPAddresses(),
	WithMACAddresses(),
)
resource, err := hostResourceDetector.Detect(context.Background())
```

## Supported attributes

According to [semantic conventions for host resources](https://opentelemetry.io/docs/specs/semconv/resource/host/), each of the following attributes is added if it is available:

* `host.arch`
* `host.id`
* `host.name`

The following attributes require an explicit opt-in during the initialization of the host resource detector:

* `host.ip`
* `host.mac`

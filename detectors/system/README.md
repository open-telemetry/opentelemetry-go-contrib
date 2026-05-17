# System Resource Detector

[![PkgGoDev](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/system)](https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/system)

A [resource detector] that collects host- and OS-level resource attributes from
the local system.

## Detected Attributes

| Attribute            | Source |
|----------------------|--------|
| `host.name`          | Configurable: DNS (FQDN), OS hostname, CNAME, or reverse DNS |
| `host.id`            | Machine ID from the OS |
| `host.arch`          | CPU architecture (`runtime.GOARCH`) |
| `host.ip`            | Non-loopback IP addresses of network interfaces |
| `host.mac`           | Non-loopback MAC addresses (IEEE RA format) |
| `host.cpu.vendor.id` | CPU vendor string via gopsutil |
| `host.cpu.family`    | CPU family via gopsutil |
| `host.cpu.model.id`  | CPU model ID via gopsutil |
| `host.cpu.model.name`| CPU model name via gopsutil |
| `host.cpu.stepping`  | CPU stepping via gopsutil |
| `host.cpu.cache.l2.size` | L2 cache size via gopsutil |
| `os.type`            | OS type (`runtime.GOOS`) |
| `os.version`         | OS platform version via gopsutil |
| `os.description`     | Human-readable OS description |

## Usage

```go
import (
    "go.opentelemetry.io/contrib/detectors/system"
    "go.opentelemetry.io/otel/sdk/resource"
)

res, err := resource.New(ctx,
    resource.WithDetectors(system.NewResourceDetector()),
)
```

### Options

**`WithHostnameSources(sources ...string)`** — Set the ordered list of
strategies for resolving `host.name`. Valid values: `"dns"` (FQDN),
`"os"` (OS hostname), `"cname"` (CNAME lookup), `"lookup"` (reverse DNS).
The first successful resolution wins. Default: `["dns", "os"]`.

```go
system.NewResourceDetector(
    system.WithHostnameSources("os"),
)
```

**`WithAttributeFilter(filter attribute.Filter)`** — Include only attributes
for which the filter returns `true`.

```go
system.NewResourceDetector(
    system.WithAttributeFilter(func(kv attribute.KeyValue) bool {
        return kv.Key == semconv.HostNameKey || kv.Key == semconv.HostIDKey
    }),
)
```

[resource detector]: https://pkg.go.dev/go.opentelemetry.io/otel/sdk/resource#Detector

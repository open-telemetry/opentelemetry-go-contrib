# Azure VM Resource detector

The Azure VM resource detector supports detecting attributes specific to Azure VMs.

## Usage

```golang
// Instantiate a new host resource detector
azureVmResourceDetector := vm.New()
resource, err := azureVmResourceDetector.Detect(context.Background())
```

## Supported attributes

According to semantic conventions for
[host](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md),
[cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md),
and
[os](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/os.md)
attributes, each of the following attributes is added if it is available:

* `cloud.provider`
* `cloud.platform`
* `host.id`
* `host.name`
* `host.type`
* `os.type`
* `os.version`

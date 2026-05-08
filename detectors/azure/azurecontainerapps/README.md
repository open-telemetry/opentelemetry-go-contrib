# Azure Container Apps Resource Detector

[![PkgGoDev](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps)](https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps)

This module detects resource attributes available in Azure Container Apps.

## Installation
```bash
go get -u go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps
```

## Usage

```go
package main

import (
    azurecontainerappsdetector "go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"
)

func main() {
    detector := azurecontainerappsdetector.NewResourceDetector()
    res, err := resource.New(ctx,
        resource.WithDetectors(detector),
    )
}
```

## Detected Attributes

| Resource Attribute | Example Value |
| --- | --- |
| `cloud.provider` | azure
|`cloud.platform` | azure.container_apps
|`service.name` | my-app
|`service.instance.id` | my-app--abc123-0
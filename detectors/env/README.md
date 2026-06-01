# Env Resource Detector

The `env` resource detector discovers resource attributes from environment variables, specifically `OTEL_RESOURCE_ATTRIBUTES` and the deprecated `OTEL_RESOURCE`.

## Usage

```go
import (
	"context"
	"go.opentelemetry.io/contrib/detectors/env"
	"go.opentelemetry.io/otel/sdk/resource"
)

func main() {
	detector := env.NewResourceDetector()
	res, err := detector.Detect(context.Background())
	if err != nil {
		// Handle error
	}
	// Use the detected resource
}
```

## Features

- Supports `OTEL_RESOURCE_ATTRIBUTES` (standard).
- Supports `OTEL_RESOURCE` (deprecated).
- Parses key=value pairs.
- Validates input format.

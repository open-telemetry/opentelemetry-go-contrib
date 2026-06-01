// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package env provides a [resource.Detector] that detects resource attributes
from the OTEL_RESOURCE_ATTRIBUTES environment variable.

The detector parses the OTEL_RESOURCE_ATTRIBUTES environment variable
as a comma-separated list of key=value pairs and adds them as resource
attributes.

[spec]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/sdk.md#specifying-resource-information-via-an-environment-variable
*/
package env // import "go.opentelemetry.io/contrib/detectors/env"

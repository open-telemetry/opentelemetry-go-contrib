# Configuration Library

This package can be used to parse a configuration file that follows the JSON
Schema defined by the [OpenTelemetry Configuration] schema.

The package contains:

- models generated via the JSON schema using the [go-jsonschema] library
- a `Create` function that interprets [configuration model] and return SDK components (TODO)
- a `Parse` function that parses and validates a [configuration file] (TODO)

## Using the generate model code

The `generated_config.go` code in versioned submodule can be used directly as-is to programmatically
produce a configuration model that can be then used as a parameter to the `Create` function. Note
that the package is versioned to match the release versioning of the opentelemetry-configuration
repository.

## Using the `Create` function (TODO)

## Using the `Parse` function (TODO)

The original code from the package comes from the [OpenTelemetry Collector's service] telemetry
configuration code. The intent being to share this code across implementations and reduce
duplication where possible.

[OpenTelemetry Configuration]: https://github.com/open-telemetry/opentelemetry-configuration/
[go-jsonschema]: https://github.com/omissis/go-jsonschema
[configuration model]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/configuration/file-configuration.md#configuration-model
[configuration file]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/configuration/file-configuration.md#configuration-file
[OpenTelemetry Collector's service]: https://github.com/open-telemetry/opentelemetry-collector/blob/7c5ecef11dff4ce5501c9683b277a25a61ea0f1a/service/telemetry/generated_config.go

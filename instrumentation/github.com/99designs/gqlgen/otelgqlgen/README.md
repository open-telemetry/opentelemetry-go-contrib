# OpenTelemetry-Go gqlgen Instrumentation

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen)

Opentelemetry instrumentation that provides middleware for gqlgen.

## Installation

```
go get -u go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen
```

## Example

See [./example](./example).

## Configuration

The instrumentation can be used with:

- Custom provider, default is global.
  [`WithTracerProvider`](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen#WithTracerProvider)
  option.
- Complexity extension, default is ComplexityLimit.
  [`WithComplexityExtensionName`](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen#WithComplexityExtensionName)
  option.

## References

- [GraphQL](https://graphql.org/)
- [gqlgen](https://gqlgen.com)
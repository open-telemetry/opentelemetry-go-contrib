# Named tracer example

[![Go Reference](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/examples/namedtracer.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib/examples/namedtracer)

Demonstrates named tracers in an application and a subpackage. It writes spans to standard output.

From the repository root:

```sh
cd examples/namedtracer
go run .
```

The output includes spans named `operation` and `Sub operation...`. The process exits after the example completes.

# Instrumentation

Code contained in this directory contains instrumentation for 3rd-party Go packages and some packages from the standard library.

## Organization

In order to ensure the maintainability and discoverability of instrumentation packages, the following guidelines MUST be followed.

### Packaging

All instrumentation packages SHOULD be of the form:

```
go.opentelemetry.io/contrib/instrumentation/{IMPORT_PATH}/otel{PACKAGE_NAME}
```

Where the [`{IMPORT_PATH}`](https://golang.org/ref/spec#ImportPath) and [`{PACKAGE_NAME}`](https://golang.org/ref/spec#PackageName) are the standard Go identifiers for the package being instrumented.

For example:

- `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`
- `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron`
- `go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql`

Exceptions to this rule do exist.
For example, the [runtime](./runtime) instrumentation does not instrument a Go package and does not fit this structure.

### Contents

All instrumentation packages MUST adhere to [the projects' contributing guidelines](../CONTRIBUTING.md).
Additionally the following guidelines for package composition need to be followed.

- All instrumentation packages MUST be a Go package.
   Therefore, an appropriately configured `go.mod` and `go.sum` need to exist for each package.
- To help understand the instrumentation a Go package documentation SHOULD be included.
   This documentation SHOULD be in a dedicated `doc.go` file if the package is more than one file.
   It SHOULD contain useful information like what the purpose of the instrumentation is, how to use it, and any compatibility restrictions that might exist. 
- Examples of how to actually use the instrumentation SHOULD be included.
- All instrumentation packages MUST provide an option to accept a `TracerProvider` if it uses a Tracer, a `MeterProvider` if it uses a Meter, and `Propagators` if it handles any context propagation.
  Also, packages MUST use the default `TracerProvider`, `MeterProvider`, and `Propagators` supplied by the `global` package if no optional one is provided.
- All instrumentation packages MUST NOT provide an option to accept a `Tracer` or `Meter`.
- All instrumentation packages MUST create any used `Tracer` or `Meter` with a name matching the instrumentation package name.

## Additional Instrumentation Packages

Below are additional instrumentation packages outside of the opentelemetry-go-contrib repo:

| Package Name | Documentation | Notes |
| :----------: | :-----------: | :---: |
| [`github.com/go-redis/redis/v8/redisext`](https://github.com/go-redis/redis/blob/v8.0.0-beta.5/redisext/otel.go) | [Go Docs](https://pkg.go.dev/github.com/go-redis/redis/v8@v8.0.0-beta.5.0.20200614113957-5b4d00c217b0/redisext?tab=doc) | Trace only; add the hook found [here](https://github.com/go-redis/redis/blob/v8.0.0-beta.5/redisext/otel.go) to your go-redis client. |

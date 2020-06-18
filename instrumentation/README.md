# Instrumentation

Code contained in this directory contains instrumentation for 3rd-party Go packages.

## Organization

In order to ensure the maintainability and discoverability of instrumentation packages, the following guidelines MUST be followed.

### Packaging

All instrumentation packages MUST be of the form:

```
go.opentelemetry.io/contrib/instrumentation/{PACKAGE}
```

Where `{PACKAGE}` is the name of the package being instrumented.

For example:

- `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux`
- `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1`
- `go.opentelemetry.io/contrib/instrumentation/database/sql`

Consequentially, this means that all instrumentation MUST be contained in a sub-directory structure matching the package name.

### Contents

All instrumentation packages MUST adhere to [the projects' contributing guidelines](../CONTRIBUTING.md).
Additionally the following guidelines for package composition need to be followed.

- All instrumentation packages MUST be a Go package.
   Therefore, an appropriately configured `go.mod` and `go.sum` need to exist for each package.
- To help understand the instrumentation a README.md SHOULD be included.
   This file SHOULD be at the top-level of the instrumentation package and contain useful information like what the instrumentation is for, how to install and use it, and any compatibility restrictions that might exist. 
- Examples of how to actually use the instrumentation SHOULD be included.

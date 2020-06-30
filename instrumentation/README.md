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
- To help understand the instrumentation a Go package documentation SHOULD be included.
   This documentation SHOULD be in a dedicated `doc.go` file if the package is more than one file.
   It SHOULD contain useful information like what the purpose of the instrumentation is, how to use it, and any compatibility restrictions that might exist. 
- Examples of how to actually use the instrumentation SHOULD be included.

## Additional Instrumentation Packages

Below are additional instrumentation packages outside of the opentelemetry-go-contrib repo:

|  Package Name  |                     Link                    |                                                          Notes                                                                 |
| :------------: | :-----------------------------------------: | :----------------------------------------------------------------------------------------------------------------------------: |
| go-redis/redis | [Github](https://github.com/go-redis/redis) | Trace only; add the hook found [here](https://github.com/go-redis/redis/blob/master/redisext/otel.go) to your go-redis client. |

# Jaeger Remote Sampler

This package implements a `"go.opentelemetry.io/otel/sdk".Sampler` that fetches its configuration from a Jaeger agent.

The sampling strategy definition: [sampling.proto](https://github.com/jaegertracing/jaeger-idl/blob/master/proto/api_v2/sampling.proto).

## Update generated Jaeger code

Files generated from jaeger-idl are checked in and usually do not have to be regenerated.

* Make sure the jaeger-idl submodule is synchronised.
  
  ```
  git submodule update jaeger-idl
  ```

*  Generate Go files from the .proto:

  ```
  make proto-gen
  ```

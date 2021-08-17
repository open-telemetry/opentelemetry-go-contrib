# Jaeger Remote Sampler

This package implements [Jaeger remote sampler](https://www.jaegertracing.io/docs/latest/sampling/#collector-sampling-configuration).

## Update generated Jaeger code

Files generated from jaeger-idl are checked in and usually do not have to be regenerated.

* Make sure the jaeger-idl submodule is synchronised.
  
  ```
  git submodule update --init jaeger-idl
  ```

*  Generate Go files from the .proto:

  ```
  make proto-gen
  ```

# Jaeger Remote Sampler

This package implements a `"go.opentelemetry.io/otel/sdk".Sampler` that fetches its configuration from a Jaeger agent.

The sampling strategy definition: [sampling.proto](https://github.com/jaegertracing/jaeger-idl/blob/master/proto/api_v2/sampling.proto).

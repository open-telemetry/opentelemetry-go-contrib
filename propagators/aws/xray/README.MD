# AWS X-Ray Propagator/IDGenerator

This package contains an AWS x-ray compatible propagator along
with an x-ray compatible IDGenerator.

## `traceIdRatioSampler` and `x-ray IDGenerator` compatability

It is a general suggestion to `NOT` use the `traceIDRatioSampler` while also
using the `x-ray IDGenerator`. The non random nature of building an x-ray traceId
may lead to inconsistent sampling results.

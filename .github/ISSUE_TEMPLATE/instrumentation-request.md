---
name: Instrumentation Request
about: Suggest instrumentation to include in this project
title: Request to Add Instrumentation for <package-name>
labels: 'enhancement, area: instrumentation'
assignees: ''

---

## Background

**Package Link**: <add link to package here>

<describe how this package is commonly used>

### Why can this instrumentation not be included in the package itself?

### Why can this instrumentation not be hosted in a dedicated repository?

## Proposed Solution

<add a high-level description of how instrumentation can wrap or hook-in to the package>

### Tracing

- attributes:
  - <add proposed attributes or remove>
- events:
  - <add proposed events or remove>
- links:
  - <add proposed links or remove>

### Metrics

Instruments

- <instrument name>: <describe what the instrument will measure>
  - type: <propose instrument type information>
  - unit: <propose instrument unit>
  - description: <propose instrument description>
  - attributes:
    - <add proposed attributes or remove>

### Prior Art

- <list other established instrumentation for this package that can be referenced>

## Tasks

- Code complete:
  - [ ] Comprehensive unit tests.
  - [ ] End-to-end integration tests.
  - [ ] Tests all passing.
  - [ ] Instrumentation functionality verified.
- Documented
  - [ ] Added to the [OpenTelemetry Registry](https://opentelemetry.io/registry/)
  - [ ] README included for the module describing high-level purpose.
  - [ ] Complete documentation of all public API including package documentation.
  - [ ] [Instrumentation documentation](https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/README.md#instrumentation-packages) updated.
- Examples
  - [ ] `Dockerfile` file to build example application.
  - [ ] `docker-compose.yml` to run example in a docker environment to demonstrate instrumentation.

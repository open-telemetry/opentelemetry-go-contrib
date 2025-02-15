# OpenTelemetry-Go Contrib

[![build_and_test](https://github.com/open-telemetry/opentelemetry-go-contrib/workflows/build_and_test/badge.svg)](https://github.com/open-telemetry/opentelemetry-go-contrib/actions?query=workflow%3Abuild_and_test+branch%3Amain)
[![codecov.io](https://codecov.io/gh/open-telemetry/opentelemetry-go-contrib/coverage.svg?branch=main)](https://app.codecov.io/gh/open-telemetry/opentelemetry-go-contrib?branch=main)
[![Docs](https://godoc.org/go.opentelemetry.io/contrib?status.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib)
[![Go Report Card](https://goreportcard.com/badge/go.opentelemetry.io/contrib)](https://goreportcard.com/report/go.opentelemetry.io/contrib)
[![Slack](https://img.shields.io/badge/slack-@cncf/otel--go-brightgreen.svg?logo=slack)](https://cloud-native.slack.com/archives/C01NPAXACKT)

Collection of 3rd-party packages for [OpenTelemetry-Go](https://github.com/open-telemetry/opentelemetry-go).

## Contents

- [Examples](./examples/): Examples of OpenTelemetry libraries usage.
- [Instrumentation](./instrumentation/): Packages providing OpenTelemetry instrumentation for 3rd-party libraries.
- [Propagators](./propagators/): Packages providing OpenTelemetry context propagators for 3rd-party propagation formats.
- [Detectors](./detectors/): Packages providing OpenTelemetry resource detectors for 3rd-party cloud computing environments.
- [Exporters](./exporters/): Packages providing OpenTelemetry exporters for 3rd-party export formats.
- [Samplers](./samplers/): Packages providing additional implementations of OpenTelemetry samplers.
- [Bridges](./bridges/): Packages providing adapters for 3rd-party instrumentation frameworks.
- [Processors](./processors/): Packages providing additional implementations of OpenTelemetry processors.

## Project Status

This project contains both stable and unstable modules.
Refer to the module for its version or our [versioning manifest](./versions.yaml).

Project versioning information and stability guarantees can be found in the [versioning documentation](https://github.com/open-telemetry/opentelemetry-go/blob/a724cf884287e04785eaa91513d26a6ef9699288/VERSIONING.md).

Progress and status specific to this repository is tracked in our local [project boards](https://github.com/open-telemetry/opentelemetry-go-contrib/projects?query=is%3Aopen) and [milestones](https://github.com/open-telemetry/opentelemetry-go-contrib/milestones).

### Compatibility

OpenTelemetry-Go Contrib ensures compatibility with the current supported
versions of
the [Go language](https://golang.org/doc/devel/release#policy):

> Each major Go release is supported until there are two newer major releases.
> For example, Go 1.5 was supported until the Go 1.7 release, and Go 1.6 was supported until the Go 1.8 release.

For versions of Go that are no longer supported upstream, opentelemetry-go-contrib will
stop ensuring compatibility with these versions in the following manner:

- A minor release of opentelemetry-go-contrib will be made to add support for the new
  supported release of Go.
- The following minor release of opentelemetry-go-contrib will remove compatibility
  testing for the oldest (now archived upstream) version of Go. This, and
  future, releases of opentelemetry-go-contrib may include features only supported by
  the currently supported versions of Go.

This project is tested on the following systems.

| OS       | Go Version | Architecture |
| -------- | ---------- | ------------ |
| Ubuntu   | 1.24       | amd64        |
| Ubuntu   | 1.23       | amd64        |
| Ubuntu   | 1.22       | amd64        |
| Ubuntu   | 1.24       | 386          |
| Ubuntu   | 1.23       | 386          |
| Ubuntu   | 1.22       | 386          |
| macOS 13 | 1.24       | amd64        |
| macOS 13 | 1.23       | amd64        |
| macOS 13 | 1.22       | amd64        |
| macOS    | 1.24       | arm64        |
| macOS    | 1.23       | arm64        |
| macOS    | 1.22       | arm64        |
| Windows  | 1.24       | amd64        |
| Windows  | 1.23       | amd64        |
| Windows  | 1.22       | amd64        |
| Windows  | 1.24       | 386          |
| Windows  | 1.23       | 386          |
| Windows  | 1.22       | 386          |

While this project should work for other systems, no compatibility guarantees
are made for those systems currently.

## Contributing

For information on how to contribute, consult [the contributing guidelines](./CONTRIBUTING.md)

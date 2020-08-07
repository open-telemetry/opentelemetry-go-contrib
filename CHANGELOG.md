# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- The `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc` module has been added to replace the instrumentation that had previoiusly existed in the `go.opentelemetry.io/otel/instrumentation/grpctrace` package. (#189)
- Instrumentation for the stdlib `net/http` and `net/http/httptrace` packages. (#190)

## [0.10.0] - 2020-07-31

This release upgrades its [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v0.10.0) dependency to v0.10.0 and includes new instrumentation for popular Kafka and Cassandra clients.

### Added

- A detector that generate resources from GCE instance. (#132)
- A detector that generate resources from AWS instances. (#139)
- Instrumentation for the Kafka client github.com/Shopify/sarama. (#134, #153)
- Links and status message for mock span in the internal testing library. (#134)
- Instrumentation for the Cassandra client github.com/gocql/gocql. (#137)
- A detector that generate resources from GKE clusters. (#154)

### Fixed

- Bump github.com/aws/aws-sdk-go from 1.33.8 to 1.33.15 in /detectors/aws. (#155, #157, #159, #162)
- Bump github.com/golangci/golangci-lint from 1.28.3 to 1.29.0 in /tools. (#146)

## [0.9.0] - 2020-07-20

This release upgrades its [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v0.9.0) dependency to v0.9.0.

### Fixed

- Bump github.com/emicklei/go-restful/v3 from 3.0.0 to 3.2.0 in /instrumentation/github.com/emicklei/go-restful. (#133)
- Update dependabot configuration to correctly check all included packages. (#131)
- Update `RELEASING.md` with correct `tag.sh` command. (#130)


## [0.8.0] - 2020-07-10

This release upgrades its [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v0.8.0) dependency to v0.8.0, includes minor fixes, and new instrumentation.

### Added

- Create this `CHANGELOG.md`. (#114)
- Add `emicklei/go-restful/v3` trace instrumentation. (#115)

### Changed

- Update `CONTRIBUTING.md` to ask for updates to `CHANGELOG.md` with each pull request. (#114)
- Move all `github.com` package instrumentation under a `github.com` directory. (#118)

### Fixed

- Update README to include information about external instrumentation.
   To start, this includes native instrumentation found in the `go-redis/redis` package. (#117)
- Bump github.com/golangci/golangci-lint from 1.27.0 to 1.28.2 in /tools. (#122, #123, #125)
- Bump go.mongodb.org/mongo-driver from 1.3.4 to 1.3.5 in /instrumentation/go.mongodb.org/mongo-driver. (#124)

## [0.7.0] - 2020-06-29

This release upgrades its [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v0.7.0) dependency to v0.7.0.

### Added

- Create `RELEASING.md` instructions. (#101)
- Apply transitive dependabot go.mod updates as part of a new automatic Github workflow. (#94)
- New dependabot integration to automate package upgrades. (#61)
- Add automatic tag generation script for release. (#60)

### Changed

- Upgrade Datadog metrics exporter to include Resource tags. (#46)
- Added output validation to Datadog example. (#96)
- Move Macaron package to match layout guidelines. (#92)
- Update top-level README and instrumentation README. (#92)
- Bump google.golang.org/grpc from 1.29.1 to 1.30.0. (#99)
- Bump github.com/golangci/golangci-lint from 1.21.0 to 1.27.0 in /tools. (#77)
- Bump go.mongodb.org/mongo-driver from 1.3.2 to 1.3.4 in /instrumentation/go.mongodb.org/mongo-driver. (#76)
- Bump github.com/stretchr/testify from 1.5.1 to 1.6.1. (#74)
- Bump gopkg.in/macaron.v1 from 1.3.5 to 1.3.9 in /instrumentation/macaron. (#68)
- Bump github.com/gin-gonic/gin from 1.6.2 to 1.6.3 in /instrumentation/gin-gonic/gin. (#73)
- Bump github.com/DataDog/datadog-go from 3.5.0+incompatible to 3.7.2+incompatible in /exporters/metric/datadog. (#78)
- Replaced `internal/trace/http.go` helpers with `api/standard` helpers from otel-go repo. (#112)

## [0.6.1] - 2020-06-08

First official tagged release of `contrib` repository.

### Added

- `labstack/echo` trace instrumentation (#42)
- `mongodb` trace instrumentation (#26)
- Go Runtime metrics (#9)
- `gorilla/mux` trace instrumentation (#19)
- `gin-gonic` trace instrumentation (#15)
- `macaron` trace instrumentation (#20)
- `dogstatsd` metrics exporter (#10)
- `datadog` metrics exporter (#22)
- Tags to all modules in repository
- Repository folder structure and automated build (#3)

### Changes

- Prefix support for dogstatsd (#34)
- Update Go Runtime package to use batch observer (#44)

[Unreleased]: https://github.com/open-telemetry/opentelemetry-go-contrib/compare/v0.10.0...HEAD
[0.10.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.10.0
[0.9.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.9.0
[0.8.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.8.0
[0.7.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.7.0
[0.6.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.6.1

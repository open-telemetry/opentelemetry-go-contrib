# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Update `CONTRIBUTING.md` to ask for updates to `CHANGELOG.md` with each pull request.

## [0.7.0] - 2020-06-29

This release upgrades its [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go "Otel Github") dependency to v0.7.0.

### Added

- Create `RELEASING.md` instructions (#101)
- Apply transitive dependabot go.mod updates as part of a new automatic Github workflow. (#94)
- New dependabot integration to automate package upgrades. (#61)
- Add automatic tag generation script for release (#60)

### Changed

- Upgrade Datadog metrics exporter to include Resource tags (#46)
- Added output validation to Datadog example (#96)
- Move Macaron package to match layout guidelines. (#92)
- Update top-level README and instrumentation README. (#92)
- Bump google.golang.org/grpc from 1.29.1 to 1.30.0 (#99)
- Bump github.com/golangci/golangci-lint from 1.21.0 to 1.27.0 in /tools (#77)
- Bump go.mongodb.org/mongo-driver from 1.3.2 to 1.3.4 in /instrumentation/go.mongodb.org/mongo-driver (#76)
- Bump github.com/stretchr/testify from 1.5.1 to 1.6.1 (#74)
- Bump gopkg.in/macaron.v1 from 1.3.5 to 1.3.9 in /instrumentation/macaron (#68)
- Bump github.com/gin-gonic/gin from 1.6.2 to 1.6.3 in /instrumentation/gin-gonic/gin (#73)
- Bump github.com/DataDog/datadog-go from 3.5.0+incompatible to 3.7.2+incompatible in /exporters/metric/datadog (#78)

### Removed

- `internal/trace/http.go` helpers, replaced by `api/standard` helpers in otel-go repo (#112)

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

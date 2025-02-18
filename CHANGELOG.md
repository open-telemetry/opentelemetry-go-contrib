# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

This release is the last to support [Go 1.22].
The next release will require at least [Go 1.23].

### Added

- Add support for configuring `ClientCertificate` and `ClientKey` field for OTLP exporters in `go.opentelemetry.io/contrib/config`. (#6378)
- Add `WithAttributeBuilder`, `AttributeBuilder`, `DefaultAttributeBuilder`, `DynamoDBAttributeBuilder`, `SNSAttributeBuilder` to support adding attributes based on SDK input and output in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#6543)
- Support for the OTEL_HTTP_CLIENT_COMPATIBILITY_MODE=http/dup environment variable in `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` to emit attributes for both the v1.20.0 and v1.26.0 semantic conventions. (#6652)
- Added the `WithMeterProvider` option to allow passing a custom meter provider to `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`. (#6648)
- Added the `WithMetricAttributesFn` option to allow setting dynamic, per-request metric attributes in `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`. (#6648)
- Added metrics support, and emit all stable metrics from the [Semantic Conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/http/http-metrics.md) in `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`. (#6648)
- Add support for configuring `Insecure` field for OTLP exporters in `go.opentelemetry.io/contrib/config`. (#6658)
- Support for the `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE=http/dup` environment variable in `instrumentation/net/http/httptrace/otelhttptrace` to emit attributes for both the v1.20.0 and v1.26.0 semantic conventions. (#6720)
- Support for the `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE=http/dup` environment variable in `instrumentation/github.com/emicklei/go-restful/otelrestful` to emit attributes for both the v1.20.0 and v1.26.0 semantic conventions. (#6710)
- Added metrics support, and emit all stable metrics from the [Semantic Conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/http/http-metrics.md) in `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`. (#6747)
- Support [Go 1.24]. (#6765)
- Add support for configuring `HeadersList` field for OTLP exporters in `go.opentelemetry.io/contrib/config`. (#6657)

### Changed

- Add custom attribute to the span after execution of the SDK rather than before in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#6543)
- Support for the `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE=http/dup` environment variable in `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` to emit attributes for both the v1.20.0 and v1.26.0 semantic conventions. (#6778)

### Deprecated

- Deprecate `WithAttributeSetter`, `AttributeSetter`, `DefaultAttributeSetter`, `DynamoDBAttributeSetter`, `SNSAttributeSetter` in favor of `WithAttributeBuilder`, `AttributeBuilder`, `DefaultAttributeBuilder`, `DynamoDBAttributeBuilder`, `SNSAttributeBuilder` in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` (#6543)

### Fixed

- Use `context.Background()` as default context instead of nil in `go.opentelemetry.io/contrib/bridges/otellogr`. (#6527)
- Convert Prometheus histogram buckets to non-cumulative otel histogram buckets in `go.opentelemetry.io/contrib/bridges/prometheus`. (#6685)
- Don't start spans that never end for filtered out gRPC stats handler in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#6695)
- Fix a possible nil dereference panic in `NewSDK` of `go.opentelemetry.io/contrib/config/v0.3.0`. (#6752)

<!-- Released section -->
<!-- Don't change this section unless doing release -->

## [1.34.0/0.59.0/0.28.0/0.14.0/0.9.0/0.7.0/0.6.0] - 2025-01-17

### Added

- Generate server metrics with semantic conventions `v1.26.0` in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` when `OTEL_SEMCONV_STABILITY_OPT_IN` is set to `http/dup`. (#6411)
- Generate client metrics with semantic conventions `v1.26.0` in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` when `OTEL_SEMCONV_STABILITY_OPT_IN` is set to `http/dup`. (#6607)

### Fixed

- Fix error logged by Jaeger remote sampler on empty or unset `OTEL_TRACES_SAMPLER_ARG` environment variable (#6511)
- Relax minimum Go version to 1.22.0 in various modules. (#6595)
- `NewSDK` handles empty `OpenTelemetryConfiguration.Resource` properly in `go.opentelemetry.io/contrib/config/v0.3.0`. (#6606)
- Fix a possible nil dereference panic in `NewSDK` of `go.opentelemetry.io/contrib/config/v0.3.0`. (#6606)

## [1.33.0/0.58.0/0.27.0/0.13.0/0.8.0/0.6.0/0.5.0] - 2024-12-12

### Added

- Added support for providing `endpoint`, `pollingIntervalMs` and `initialSamplingRate` using environment variable `OTEL_TRACES_SAMPLER_ARG` in `go.opentelemetry.io/contrib/samples/jaegerremote`. (#6310)
- Added support exporting logs via OTLP over gRPC in `go.opentelemetry.io/contrib/config`. (#6340)
- The `go.opentelemetry.io/contrib/bridges/otellogr` module.
  This module provides an OpenTelemetry logging bridge for `github.com/go-logr/logr`. (#6386)
- Added SNS instrumentation in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#6388)
- Use a `sync.Pool` for metric options in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#6394)
- Added support for configuring `Certificate` field when configuring OTLP exporters in `go.opentelemetry.io/contrib/config`. (#6376)
- Added support for the `WithMetricAttributesFn` option to middlewares in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#6542)

### Changed

- Change the span name to be `GET /path` so it complies with the OTel HTTP semantic conventions in `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho`. (#6365)
- Record errors instead of setting the `gin.errors` attribute in `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`. (#6346)
- The `go.opentelemetry.io/contrib/config` now supports multiple schemas in subdirectories (i.e. `go.opentelemetry.io/contrib/config/v0.3.0`) for easier migration. (#6412)

### Fixed

- Fix broken AWS presigned URLs when using instrumentation in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#5975)
- Fixed the value for configuring the OTLP exporter to use `grpc` instead of `grpc/protobuf` in `go.opentelemetry.io/contrib/config`. (#6338)
- Allow marshaling types in `go.opentelemetry.io/contrib/config`. (#6347)
- Removed the redundant handling of panic from the `HTML` function in `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`. (#6373)
- The `code.function` attribute emitted by `go.opentelemetry.io/contrib/bridges/otelslog` now stores just the function name instead the package path-qualified function name.
  The `code.namespace` attribute now stores the package path. (#6415)
- The `code.function` attribute emitted by `go.opentelemetry.io/contrib/bridges/otelzap` now stores just the function name instead the package path-qualified function name.
  The `code.namespace` attribute now stores the package path. (#6423)
- Return an error for `nil` values when unmarshaling `NameStringValuePair` in `go.opentelemetry.io/contrib/config`. (#6425)

## [1.32.0/0.57.0/0.26.0/0.12.0/0.7.0/0.5.0/0.4.0] - 2024-11-08

### Added

- Add the `WithSource` option to the `go.opentelemetry.io/contrib/bridges/otelslog` log bridge to set the `code.*` attributes in the log record that includes the source location where the record was emitted. (#6253)
- Add `ContextWithStartTime` and `StartTimeFromContext` to `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, which allows setting the start time using go context. (#6137)
- Set the `code.*` attributes in `go.opentelemetry.io/contrib/bridges/otelzap` if the `zap.Logger` was created with the `AddCaller` or `AddStacktrace` option. (#6268)
- Add a `LogProcessor` to `go.opentelemetry.io/contrib/processors/baggagecopy` to copy baggage members to log records. (#6277)
  - Use `baggagecopy.NewLogProcessor` when configuring a Log Provider.
    - `NewLogProcessor` accepts a `Filter` function type that selects which baggage members are added to the log record.

### Changed

- Transform raw (`slog.KindAny`) attribute values to matching `log.Value` types.
  For example, `[]string{"foo", "bar"}` attribute value is now transformed to `log.SliceValue(log.StringValue("foo"), log.StringValue("bar"))` instead of `log.String("[foo bar"])`. (#6254)
- Upgrade `go.opentelemetry.io/otel/semconv/v1.17.0` to `go.opentelemetry.io/otel/semconv/v1.21.0` in `go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo`. (#6272)
- Resource doesn't merge with defaults if a valid resource is configured in `go.opentelemetry.io/contrib/config`. (#6289)

### Fixed

- Transform nil attribute values to `log.Value` zero value instead of panicking in `go.opentelemetry.io/contrib/bridges/otellogrus`. (#6237)
- Transform nil attribute values to `log.Value` zero value instead of panicking in `go.opentelemetry.io/contrib/bridges/otelzap`. (#6237)
- Transform nil attribute values to `log.Value` zero value instead of `log.StringValue("<nil>")` in `go.opentelemetry.io/contrib/bridges/otelslog`. (#6246)
- Fix `NewClientHandler` so that `rpc.client.request.*` metrics measure requests instead of responses and `rpc.client.responses.*` metrics measure responses instead of requests in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#6250)
- Fix issue in `go.opentelemetry.io/contrib/config` causing `otelprom.WithResourceAsConstantLabels` configuration to not be respected. (#6260)
- `otel.Handle` is no longer called on a successful shutdown of the Prometheus exporter in `go.opentelemetry.io/contrib/config`. (#6299)

## [1.31.0/0.56.0/0.25.0/0.11.0/0.6.0/0.4.0/0.3.0] - 2024-10-14

### Added

- The `Severitier` and `SeverityVar` types are added to `go.opentelemetry.io/contrib/processors/minsev` allowing dynamic configuration of the severity used by the `LogProcessor`. (#6116)
- Move examples from `go.opentelemetry.io/otel` to this repository under `examples` directory. (#6158)
- Support yaml/json struct tags for generated code in `go.opentelemetry.io/contrib/config`. (#5433)
- Add support for parsing YAML configuration via `ParseYAML` in `go.opentelemetry.io/contrib/config`. (#5433)
- Add support for temporality preference configuration in `go.opentelemetry.io/contrib/config`. (#5860)

### Changed

- The function signature of `NewLogProcessor` in `go.opentelemetry.io/contrib/processors/minsev` has changed to accept the added `Severitier` interface instead of a `log.Severity`. (#6116)
- Updated `go.opentelemetry.io/contrib/config` to use the [v0.3.0](https://github.com/open-telemetry/opentelemetry-configuration/releases/tag/v0.3.0) release of schema which includes backwards incompatible changes. (#6126)
- `NewSDK` in `go.opentelemetry.io/contrib/config` now returns a no-op SDK if `disabled` is set to `true`. (#6185)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho` package has found a Code Owner.
  The package is no longer deprecated. (#6207)

### Fixed

- Possible nil dereference panic in `go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace`. (#5965)
- `logrus.Level` transformed to appropriate `log.Severity` in `go.opentelemetry.io/contrib/bridges/otellogrus`. (#6191)

### Removed

- The `Minimum` field of the `LogProcessor` in `go.opentelemetry.io/contrib/processors/minsev` is removed.
  Use `NewLogProcessor` to configure this setting. (#6116)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron` package is removed. (#6186)
- The deprecated `go.opentelemetry.io/contrib/samplers/aws/xray` package is removed. (#6187)

## [1.30.0/0.55.0/0.24.0/0.10.0/0.5.0/0.3.0/0.2.0] - 2024-09-10

### Added

- Add `NewProducer` to `go.opentelemetry.io/contrib/instrumentation/runtime`, which allows collecting the `go.schedule.duration` histogram metric from the Go runtime. (#5991)
- Add gRPC protocol support for OTLP log exporter in `go.opentelemetry.io/contrib/exporters/autoexport`. (#6083)

### Removed

- Drop support for [Go 1.21]. (#6046, #6047)

### Fixed

- Superfluous call to `WriteHeader` when flushing after setting a status code in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#6074)
- Superfluous call to `WriteHeader` when writing the response body after setting a status code in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#6055)

## [1.29.0/0.54.0/0.23.0/0.9.0/0.4.0/0.2.0/0.1.0] - 2024-08-23

This release is the last to support [Go 1.21].
The next release will require at least [Go 1.22].

### Added

- Add the `WithSpanAttributes` and `WithMetricAttributes` methods to set custom attributes to the stats handler in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#5133)
- The `go.opentelemetry.io/contrib/bridges/otelzap` module.
  This module provides an OpenTelemetry logging bridge for `go.uber.org/zap`. (#5191)
- Support for the `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE=http/dup` environment variable in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` to emit attributes for both the v1.20.0 and v1.26.0 semantic conventions. (#5401)
- The `go.opentelemetry.io/contrib/bridges/otelzerolog` module.
  This module provides an OpenTelemetry logging bridge for `github.com/rs/zerolog`. (#5405)
- Add `WithGinFilter` filter parameter in `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` to allow filtering requests with `*gin.Context`. (#5743)
- Support for stdoutlog exporter in `go.opentelemetry.io/contrib/config`. (#5850)
- Add macOS ARM64 platform to the compatibility testing suite. (#5868)
- Add new runtime metrics to `go.opentelemetry.io/contrib/instrumentation/runtime`, which are still disabled by default. (#5870)
- Add the `WithMetricsAttributesFn` option to allow setting dynamic, per-request metric attributes in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#5876)
- The `go.opentelemetry.io/contrib/config` package supports configuring `with_resource_constant_labels` for the prometheus exporter. (#5890)
- Support [Go 1.23]. (#6017)

### Removed

- The deprecated `go.opentelemetry.io/contrib/processors/baggagecopy` package is removed. (#5853)

### Fixed

- Race condition when reading the HTTP body and writing the response in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#5916)

## [1.28.0/0.53.0/0.22.0/0.8.0/0.3.0/0.1.0] - 2024-07-02

### Added

- Add the new `go.opentelemetry.io/contrib/detectors/azure/azurevm` package to provide a resource detector for Azure VMs. (#5422)
- Add support to configure views when creating MeterProvider using the config package. (#5654)
- The `go.opentelemetry.io/contrib/config` add support to configure periodic reader interval and timeout. (#5661)
- Add log support for the autoexport package. (#5733)
- Add support for disabling the old runtime metrics using the `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=false` environment variable. (#5747)
- Add support for signal-specific protocols environment variables (`OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`, `OTEL_EXPORTER_OTLP_LOGS_PROTOCOL`, `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL`) in `go.opentelemetry.io/contrib/exporters/autoexport`. (#5816)
- The `go.opentelemetry.io/contrib/processors/minsev` module is added.
  This module provides and experimental logging processor with a configurable threshold for the minimum severity records must have to be recorded. (#5817)
- The `go.opentelemetry.io/contrib/processors/baggagecopy` module.
  This module is a replacement of `go.opentelemetry.io/contrib/processors/baggage/baggagetrace`. (#5824)

### Changed

- Improve performance of `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` with the usage of `WithAttributeSet()` instead of `WithAttribute()`. (#5664)
- Improve performance of `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` with the usage of `WithAttributeSet()` instead of `WithAttribute()`. (#5664)
- Update `go.opentelemetry.io/contrib/config` to latest released configuration schema which introduces breaking changes where `Attributes` is now a `map[string]interface{}`. (#5758)
- Upgrade all dependencies of `go.opentelemetry.io/otel/semconv/v1.25.0` to `go.opentelemetry.io/otel/semconv/v1.26.0`. (#5847)

### Fixed

- Custom attributes targeting metrics recorded by the `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` are not ignored anymore. (#5129)
- The double setup in `go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/example` that caused duplicate traces. (#5564)
- The superfluous `response.WriteHeader` call in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` when the response writer is flushed. (#5634)
- Use `c.FullPath()` method to set `http.route` attribute in `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`. (#5734)
- Out-of-bounds panic in case of invalid span ID in `go.opentelemetry.io/contrib/propagators/b3`. (#5754)

### Deprecated

- The `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho` package is deprecated.
  If you would like to become a Code Owner of this module and prevent it from being removed, see [#5550]. (#5645)
- The `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron` package is deprecated.
  If you would like to become a Code Owner of this module and prevent it from being removed, see [#5552]. (#5646)
- The `go.opentelemetry.io/contrib/samplers/aws/xray` package is deprecated.
  If you would like to become a Code Owner of this module and prevent it from being removed, see [#5554]. (#5647)
- The `go.opentelemetry.io/contrib/processors/baggage/baggagetrace` package is deprecated.
  Use the added `go.opentelemetry.io/contrib/processors/baggagecopy` package instead. (#5824)
  - Use `baggagecopy.NewSpanProcessor` as a replacement for `baggagetrace.New`.
    - `NewSpanProcessor` accepts a `Filter` function type that selects which baggage members are added to a span.
    - `NewSpanProcessor` returns a `*baggagecopy.SpanProcessor` instead of a `trace.SpanProcessor` interface.
      The returned type still implements the interface.

[#5550]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5550
[#5552]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5552
[#5554]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5554

## [1.27.0/0.52.0/0.21.0/0.7.0/0.2.0] - 2024-05-21

### Added

- Add an experimental `OTEL_METRICS_PRODUCERS` environment variable to `go.opentelemetry.io/contrib/autoexport` to be set metrics producers. (#5281)
  - `prometheus` and `none` are supported values. You can specify multiple producers separated by a comma.
  - Add `WithFallbackMetricProducer` option that adds a fallback if the `OTEL_METRICS_PRODUCERS` is not set or empty.
- The `go.opentelemetry.io/contrib/processors/baggage/baggagetrace` module. This module provides a Baggage Span Processor. (#5404)
- Add gRPC trace `Filter` for stats handler to `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#5196)
- Add a repository Code Ownership Policy. (#5555)
- The `go.opentelemetry.io/contrib/bridges/otellogrus` module.
  This module provides an OpenTelemetry logging bridge for `github.com/sirupsen/logrus`. (#5355)
- The `WithVersion` option function in `go.opentelemetry.io/contrib/bridges/otelslog`.
  This option function is used as a replacement of `WithInstrumentationScope` to specify the logged package version. (#5588)
- The `WithSchemaURL` option function in `go.opentelemetry.io/contrib/bridges/otelslog`.
  This option function is used as a replacement of `WithInstrumentationScope` to specify the semantic convention schema URL for the logged records. (#5588)
- Add support for Cloud Run jobs in `go.opentelemetry.io/contrib/detectors/gcp`. (#5559)

### Changed

- The gRPC trace `Filter` for interceptor is renamed to `InterceptorFilter`. (#5196)
- The gRPC trace filter functions `Any`, `All`, `None`, `Not`, `MethodName`, `MethodPrefix`, `FullMethodName`, `ServiceName`, `ServicePrefix` and `HealthCheck` for interceptor are moved to `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters/interceptor`.
  With this change, the filters in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` are now working for stats handler. (#5196)
- `NewSDK` in `go.opentelemetry.io/contrib/config` now returns a configured SDK with a valid `LoggerProvider`. (#5427)

- `NewLogger` now accepts a `name` `string` as the first argument.
  This parameter is used as a replacement of `WithInstrumentationScope` to specify the name of the logger backing the underlying `Handler`. (#5588)
- `NewHandler` now accepts a `name` `string` as the first argument.
  This parameter is used as a replacement of `WithInstrumentationScope` to specify the name of the logger backing the returned `Handler`. (#5588)
- Upgrade all dependencies of `go.opentelemetry.io/otel/semconv/v1.24.0` to `go.opentelemetry.io/otel/semconv/v1.25.0`. (#5605)

### Removed

- The `WithInstrumentationScope` option function in `go.opentelemetry.io/contrib/bridges/otelslog` is removed.
  Use the `name` parameter added to `NewHandler` and `NewLogger` as well as `WithVersion` and `WithSchema` as replacements. (#5588)

### Deprecated

- The `InterceptorFilter` type in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` is deprecated. (#5196)

## [1.26.0/0.51.0/0.20.0/0.6.0/0.1.0] - 2024-04-24

### Added

- `NewSDK` in `go.opentelemetry.io/contrib/config` now returns a configured SDK with a valid `MeterProvider`. (#4804)

### Changed

- Change the scope name for the prometheus bridge to `go.opentelemetry.io/contrib/bridges/prometheus` to match the package. (#5396)
- Add support for settings additional properties for resource configuration in `go.opentelemetry.io/contrib/config`. (#4832)

### Fixed

- Fix bug where an empty exemplar was added to counters in `go.opentelemetry.io/contrib/bridges/prometheus`. (#5395)
- Fix bug where the last histogram bucket was missing in `go.opentelemetry.io/contrib/bridges/prometheus`. (#5395)

## [1.25.0/0.50.0/0.19.0/0.5.0/0.0.1] - 2024-04-05

### Added

- Implemented setting the `cloud.resource_id` resource attribute in `go.opentelemetry.io/detectors/aws/ecs` based on the ECS Metadata v4 endpoint. (#5091)
- The `go.opentelemetry.io/contrib/bridges/otelslog` module.
  This module provides an OpenTelemetry logging bridge for "log/slog". (#5335)

### Fixed

- Update all dependencies to address [GO-2024-2687]. (#5359)

### Removed

- Drop support for [Go 1.20]. (#5163)

## [1.24.0/0.49.0/0.18.0/0.4.0] - 2024-02-23

This release is the last to support [Go 1.20].
The next release will require at least [Go 1.21].

### Added

- Support [Go 1.22]. (#5082)
- Add support for Summary metrics to `go.opentelemetry.io/contrib/bridges/prometheus`. (#5089)
- Add support for Exponential (native) Histograms in `go.opentelemetry.io/contrib/bridges/prometheus`. (#5093)

### Removed

- The deprecated `RequestCount` constant in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is removed. (#4894)
- The deprecated `RequestContentLength` constant in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is removed. (#4894)
- The deprecated `ResponseContentLength` constant in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is removed. (#4894)
- The deprecated `ServerLatency` constant in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is removed. (#4894)

### Fixed

- Retrieving the body bytes count in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` does not cause a data race anymore. (#5080)

## [1.23.0/0.48.0/0.17.0/0.3.0] - 2024-02-06

### Added

- Add client metric support to `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#4707)
- Add peer attributes to spans recorded by `NewClientHandler`, `NewServerHandler` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#4873)
- Add support for `cloud.account.id`, `cloud.availability_zone` and `cloud.region` in the AWS ECS detector. (#4860)

### Changed

- The fallback options in  `go.opentelemetry.io/contrib/exporters/autoexport` now accept factory functions. (#4891)
  - `WithFallbackMetricReader(metric.Reader) MetricOption` is replaced with `func WithFallbackMetricReader(func(context.Context) (metric.Reader, error)) MetricOption`.
  - `WithFallbackSpanExporter(trace.SpanExporter) SpanOption` is replaced with `WithFallbackSpanExporter(func(context.Context) (trace.SpanExporter, error)) SpanOption`.
- The `http.server.request_content_length` metric in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is changed to `http.server.request.size`.(#4707)
- The `http.server.response_content_length` metric in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is changed to `http.server.response.size`.(#4707)

### Deprecated

- The `RequestCount`, `RequestContentLength`, `ResponseContentLength`, `ServerLatency` constants in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` are deprecated. (#4707)

### Fixed

- Do not panic in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` if `MeterProvider` returns a `nil` instrument. (#4875)

## [1.22.0/0.47.0/0.16.0/0.2.0] - 2024-01-18

### Added

- Add `SDK.Shutdown` method in `"go.opentelemetry.io/contrib/config"`. (#4583)
- `NewSDK` in `go.opentelemetry.io/contrib/config` now returns a configured SDK with a valid `TracerProvider`. (#4741)

### Changed

- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/example` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example` are upgraded to v1.20.0. (#4320)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`are upgraded to v1.20.0. (#4320)
- Updated configuration schema to include `schema_url` for resource definition and `without_type_suffix` and `without_units` for the Prometheus exporter. (#4727)
- The semantic conventions used by the `go.opentelemetry.io/contrib/detectors/aws/ecs` resource detector are upgraded to v1.24.0. (#4803)
- The semantic conventions used by the `go.opentelemetry.io/contrib/detectors/aws/lambda` resource detector are upgraded to v1.24.0. (#4803)
- The semantic conventions used by the `go.opentelemetry.io/contrib/detectors/aws/ec2` resource detector are upgraded to v1.24.0. (#4803)
- The semantic conventions used by the `go.opentelemetry.io/contrib/detectors/aws/eks` resource detector are upgraded to v1.24.0. (#4803)
- The semantic conventions used by the `go.opentelemetry.io/contrib/detectors/gcp` resource detector are upgraded to v1.24.0. (#4803)
- The semantic conventions used in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/test` are upgraded to v1.24.0. (#4803)

### Fixed

- Fix `NewServerHandler` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` to correctly set the span status depending on the gRPC status. (#4587)
- The `stats.Handler` from `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` now does not crash when receiving an unexpected context. (#4825)
- Update `go.opentelemetry.io/contrib/detectors/aws/ecs` to fix the task ARN when it is not valid. (#3583)
- Do not panic in `go.opentelemetry.io/contrib/detectors/aws/ecs` when the container ARN is not valid. (#3583)

## [1.21.1/0.46.1/0.15.1/0.1.1] - 2023-11-16

### Changed

- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.21.0`/`v0.44.0` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.21.0). (#4582)

### Fixed

- Fix `StreamClientInterceptor` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` to end the spans synchronously. (#4537)
- Fix data race in stats handlers when processing messages received and sent metrics in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#4577)
- The stats handlers `NewClientHandler`, `NewServerHandler` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` now record RPC durations in `ms` instead of `ns`. (#4548)

## [1.21.0/0.46.0/0.15.0/0.1.0] - 2023-11-10

### Added

- Add `"go.opentelemetry.io/contrib/samplers/jaegerremote".WithSamplingStrategyFetcher` which sets custom fetcher implementation. (#4045)
- Add `"go.opentelemetry.io/contrib/config"` package that includes configuration models generated via go-jsonschema. (#4376)
- Add `NewSDK` function to `"go.opentelemetry.io/contrib/config"`. The initial implementation only returns noop providers. (#4414)
- Add metrics support (No-op, OTLP and Prometheus) to `go.opentelemetry.io/contrib/exporters/autoexport`. (#4229, #4479)
- Add support for `console` span exporter and metrics exporter in `go.opentelemetry.io/contrib/exporters/autoexport`. (#4486)
- Set unit and description on all instruments in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#4500)
- Add metric support for `grpc.StatsHandler` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#4356)
- Expose the name of the scopes in all instrumentation libraries as `ScopeName`. (#4448)

### Changed

- Dropped compatibility testing for [Go 1.19].
  The project no longer guarantees support for this version of Go. (#4352)
- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.20.0`/`v0.43.0` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.20.0). (#4546)
- In `go.opentelemetry.io/contrib/exporters/autoexport`, `Option` was renamed to `SpanOption`. The old name is deprecated but continues to be supported as an alias. (#4229)

### Deprecated

- The interceptors (`UnaryClientInterceptor`, `StreamClientInterceptor`, `UnaryServerInterceptor`, `StreamServerInterceptor`, `WithInterceptorFilter`) are deprecated. Use stats handlers (`NewClientHandler`, `NewServerHandler`) instead. (#4534)

### Fixed

- The `go.opentelemetry.io/contrib/samplers/jaegerremote` sampler does not panic when the default HTTP round-tripper (`http.DefaultTransport`) is not `*http.Transport`. (#4045)
- The `UnaryServerInterceptor` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` now sets gRPC status code correctly for the `rpc.server.duration` metric. (#4481)
- The `NewClientHandler`, `NewServerHandler` in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` now honor `otelgrpc.WithMessageEvents` options. (#4536)
- The `net.sock.peer.*` and `net.peer.*` high cardinality attributes are removed from the metrics generated by `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#4322)

## [1.20.0/0.45.0/0.14.0] - 2023-09-28

### Added

- Set the description for the `rpc.server.duration` metric in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#4302)
- Add `NewServerHandler` and `NewClientHandler` that return a `grpc.StatsHandler` used for gRPC instrumentation in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#3002)
- Add new Prometheus bridge module in `go.opentelemetry.io/contrib/bridges/prometheus`. (#4227)

### Changed

- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.19.0`/`v0.42.0`/`v0.0.7` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.19.0).
- Use `grpc.StatsHandler` for gRPC instrumentation in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/example`. (#4325)

## [1.19.0/0.44.0/0.13.0] - 2023-09-12

### Added

- Add `gcp.gce.instance.name` and `gcp.gce.instance.hostname` resource attributes to `go.opentelemetry.io/contrib/detectors/gcp`. (#4263)

### Changed

- The semantic conventions used by `go.opentelemetry.io/contrib/detectors/aws/ec2` have been upgraded to v1.21.0. (#4265)
- The semantic conventions used by `go.opentelemetry.io/contrib/detectors/aws/ecs` have been upgraded to v1.21.0. (#4265)
- The semantic conventions used by `go.opentelemetry.io/contrib/detectors/aws/eks` have been upgraded to v1.21.0. (#4265)
- The semantic conventions used by `go.opentelemetry.io/contrib/detectors/aws/lambda` have been upgraded to v1.21.0. (#4265)
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda` have been upgraded to v1.21.0. (#4265)
  - The `faas.execution` attribute is now `faas.invocation_id`.
  - The `faas.id` attribute is now `aws.lambda.invoked_arn`.
- The semantic conventions used by `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` have been upgraded to v1.21.0. (#4265)
- The `http.request.method` attribute will only allow known HTTP methods from the metrics generated by `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#4277)

### Removed

- The high cardinality attributes `net.sock.peer.addr`, `net.sock.peer.port`, `http.user_agent`, `enduser.id`, and `http.client_ip` were removed from the metrics generated by `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#4277)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego` module is removed. (#4295)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit` module is removed. (#4295)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama` module is removed. (#4295)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache` module is removed. (#4295)
- The deprecated `go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql` module is removed. (#4295)

## [1.18.0/0.43.0/0.12.0] - 2023-08-28

### Added

- Add `NewMiddleware` function in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#2964)
- The `go.opentelemetry.io/contrib/exporters/autoexport` package to provide configuration of trace exporters with useful defaults and environment variable support. (#2753, #4100, #4130, #4132, #4134)
- `WithRouteTag` in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` adds HTTP route attribute to metrics. (#615)
- Add `WithSpanOptions` option in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#3768)
- Add testing support for Go 1.21. (#4233)
- Add `WithFilter` option to `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`. (#4230)

### Changed

- Change interceptors in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` to disable `SENT`/`RECEIVED` events.
  Use `WithMessageEvents()` to turn back on. (#3964)

### Changed

- `go.opentelemetry.io/contrib/detectors/gcp`: Detect `faas.instance` instead of `faas.id`, since `faas.id` is being removed. (#4198)

### Fixed

- AWS XRay Remote Sampling to cap `quotaBalance` to 1x quota in `go.opentelemetry.io/contrib/samplers/aws/xray`. (#3651, #3652)
- Do not panic when the HTTP request has the "Expect: 100-continue" header in `go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace`. (#3892)
- Fix span status value set for non-standard HTTP status codes in modules listed below. (#3966)
  - `go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful`
  - `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`
  - `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`
  - `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho`
  - `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron`
  - `go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace`
  - `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- Do not modify the origin request in `RoundTripper` in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#4033)
- Handle empty value of `OTEL_PROPAGATORS` environment variable the same way as when the variable is unset in `go.opentelemetry.io/contrib/propagators/autoprop`. (#4101)
- Fix gRPC service/method URL path parsing discrepancies in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#4135)

### Deprecated

- The `go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego` module is deprecated. (#4092, #4104)
- The `go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit` module is deprecated. (#4093, #4104)
- The `go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama` module is deprecated. (#4099)
- The `go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache` module is deprecated. (#4164)
- The `go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql` module is deprecated. (#4164)

### Removed

- Remove `Handler` type in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#2964)

## [1.17.0/0.42.0/0.11.0] - 2023-05-23

### Changed

- Use `strings.Cut()` instead of `string.SplitN()` for better readability and memory use. (#3822)

## [1.17.0-rc.1/0.42.0-rc.1/0.11.0-rc.1] - 2023-05-17

### Changed

- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.16.0-rc.1`/`v0.39.0-rc.1` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.16.0-rc.1).
- Remove `semver:` prefix from instrumentation version. (#3681, #3798)

### Deprecated

- `SemVersion` functions in instrumentation packages are deprecated, use `Version` instead. (#3681, #3798)

## [1.16.1/0.41.1/0.10.1] - 2023-05-02

### Added

- The `WithPublicEndpoint` and `WithPublicEndpointFn` options in `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`. (#3661)

### Changed

- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.15.1`/`v0.38.1` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.15.1)

### Fixed

- AWS XRay Remote Sampling to preserve previous rule if updated rule property has not changed in `go.opentelemetry.io/contrib/samplers/aws/xray`. (#3619, #3620)

## [1.16.0/0.41.0/0.10.0] - 2023-04-28

### Added

- AWS SDK add `rpc.system` attribute in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#3582, #3617)

### Changed

- Update `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` to align gRPC server span status with the changes in the OpenTelemetry specification. (#3685)
- Adding the `db.statement` tag to spans in `go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo` is now disabled by default. (#3519)

### Fixed

- The error received by `otelecho` middleware is then passed back to upstream middleware instead of being swallowed. (#3656)
- Prevent taking from reservoir in AWS XRay Remote Sampler when there is zero capacity in `go.opentelemetry.io/contrib/samplers/aws/xray`. (#3684)
- Fix `otelhttp.Handler` in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` to propagate multiple `WriteHeader` calls while persisting the initial `statusCode`. (#3580)

## [1.16.0-rc.2/0.41.0-rc.2/0.10.0-rc.2] - 2023-03-23

### Added

- The `WithPublicEndpoint` and `WithPublicEndpointFn` options in `go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful`. (#3563)

### Fixed

- AWS SDK rename attributes `aws.operation`, `aws.service` to `rpc.method`,`rpc.service` in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#3582, #3617)
- AWS SDK span name to be of the format `Service.Operation` in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#3582, #3521)
- Prevent sampler configuration reset from erroneously sampling first span in `go.opentelemetry.io/contrib/samplers/jaegerremote`. (#3603, #3604)

## [1.16.0-rc.1/0.41.0-rc.1/0.10.0-rc.1] - 2023-03-02

### Changed

- Dropped compatibility testing for [Go 1.18].
  The project no longer guarantees support for this version of Go. (#3516)

## [1.15.0/0.40.0/0.9.0] - 2023-02-27

This release is the last to support [Go 1.18].
The next release will require at least [Go 1.19].

### Added

- Support [Go 1.20]. (#3372)
- Add `SpanNameFormatter` option to package `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`. (#3343)

### Changed

- Change to use protobuf parser instead of encoding/json to accept enums as strings in `go.opentelemetry.io/contrib/samplers/jaegerremote`. (#3183)

### Fixed

- Remove use of deprecated `"math/rand".Seed` in `go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama/example/producer`. (#3396)
- Do not assume "aws" partition in ecs detector to prevent panic in `go.opentelemetry.io/contrib/detectors/aws/ecs`. (#3167)
- The span name of producer spans from `go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama` is corrected to use `publish` instead of `send`. (#3369)
- Attribute types are corrected in `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`. (#3369)
  - `aws.dynamodb.table_names` is now a string slice value.
  - `aws.dynamodb.global_secondary_indexes` is now a string slice value.
  - `aws.dynamodb.local_secondary_indexes` is now a string slice value.
  - `aws.dynamodb.attribute_definitions` is now a string slice value.
  - `aws.dynamodb.global_secondary_index_updates` is now a string slice value.
  - `aws.dynamodb.provisioned_read_capacity` is now a `float64` value.
  - `aws.dynamodb.provisioned_write_capacity` is now a `float64` value.

## [1.14.0/0.39.0/0.8.0] - 2023-02-07

### Changed

- Change `runtime.uptime` instrument in `go.opentelemetry.io/contrib/instrumentation/runtime` from `Int64ObservableUpDownCounter` to `Int64ObservableCounter`,
 since the value is monotonic. (#3347)
- `samplers/jaegerremote`: change to use protobuf parser instead of encoding/json to accept enums as strings. (#3183)

### Fixed

- The GCE detector in `go.opentelemetry.io/contrib/detectors/gcp` includes the "cloud.region" attribute when appropriate. (#3367)

## [1.13.0/0.38.0/0.7.0] - 2023-01-30

### Added

- Add `WithSpanNameFormatter` to `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` to allow customizing span names. (#3041)
- Add missing recommended AWS Lambda resource attributes `faas.instance` and `faas.max_memory` in `go.opentelemetry.io/contrib/detectors/aws/lambda`. (#3148)
- Improve documentation for `go.opentelemetry.io/contrib/samplers/jaegerremote` by providing examples of sampling endpoints. (#3147)
- Add `WithServerName` to `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` to set the primary server name of a `Handler`. (#3182)

### Changed

- Remove expensive calculation of uncompressed message size attribute in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#3168)
- Upgrade all `semconv` packages to use `v1.17.0`. (#3182)
- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.12.0`/`v0.35.0` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.12.0). (#3190, #3170)

## [1.12.0/0.37.0/0.6.0]

### Added

- Implemented retrieving the [`aws.ecs.*` resource attributes](https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/cloud_provider/aws/ecs/) in `go.opentelemetry.io/detectors/aws/ecs` based on the ECS Metadata v4 endpoint. (#2626)
- The `WithLogger` option to `go.opentelemetry.io/contrib/samplers/jaegerremote` to allow users to pass a `logr.Logger` and have operations logged. (#2566)
- Add the `messaging.url` & `messaging.system` attributes to all appropriate SQS operations in the `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` package. (#2879)
- Add example use of the metrics signal to `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/example`. (#2610)
- [otelgin] Add support for filters to the `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` package to provide the way to control which inbound requests are traced. (#2965, #2963)

### Fixed

- Set the status_code span attribute even if the HTTP handler hasn't written anything. (#2822)
- Do not wrap http.NoBody in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, which fixes handling of that special request body. (#2983)

## [1.11.1/0.36.4/0.5.2]

### Added

- Add trace context propagation support to `instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` (#2856).
- [otelgrpc] Add `WithMeterProvider` function to enable metric and add metric `rpc.server.duration` to otelgrpc instrumentation library. (#2700)

### Changed

- Upgrade dependencies of OpenTelemetry Go to use the new [`v1.11.1`/`v0.33.0` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.11.1)

## [1.11.0/0.36.3/0.5.1]

### Changed

- Upgrade dependencies of the OpenTelemetry Go Metric SDK to use the new [`v1.11.0`/`v0.32.3` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.11.0)

## [0.36.2]

### Changed

- Upgrade dependencies of the OpenTelemetry Go Metric SDK to use the new [`v0.32.2` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/sdk%2Fmetric%2Fv0.32.2)
- Avoid getting a new Tracer for every RPC in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#2835)
- Conditionally compute message size for tracing events using proto v2 API rather than legacy v1 API in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`. (#2647)

### Deprecated

- The `Inject` function in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` is deprecated. (#2838)
- The `Extract` function in `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` is deprecated. (#2838)

## [0.36.1]

### Changed

- Upgrade dependencies of the OpenTelemetry Go Metric SDK to use the new [`v0.32.1` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/sdk%2Fmetric%2Fv0.32.1)

### Removed

- Drop support for Go 1.17.
  The project currently only supports Go 1.18 and above. (#2785)

## [0.36.0]

### Changed

- Upgrade dependencies of the OpenTelemetry Go Metric SDK to use the new [`v0.32.0` release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/sdk%2Fmetric%2Fv0.32.0). (#2781, #2756, #2758, #2760, #2762)

## [1.10.0/0.35.0/0.5.0]

### Changed

- Rename the `Typ` field of `"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc".InterceptorInfo` to `Type`. (#2688)
- Use Go 1.19 as the default version for CI testing/linting. (#2675)

### Fixed

- Fix the Jaeger propagator rejecting trace IDs that are both shorter than 128 bits and not exactly 64 bits long (while not being 0).
  Also fix the propagator rejecting span IDs shorter than 64 bits.
  This fixes compatibility with Jaeger clients encoding trace and span IDs as variable-length hex strings, [as required by the Jaeger propagation format](https://www.jaegertracing.io/docs/1.37/client-libraries/#value). (#2731)

## [1.9.0/0.34.0/0.4.0] - 2022-08-02

### Added

- Add gRPC trace `Filter` to the `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` package to provide the way to filter the traces automatically generated in interceptors. (#2572)
- The `TextMapPropagator` function to `go.opentelemetry.io/contrib/propagators/autoprop`.
  This function is used to return a composite `TextMapPropagator` from registered names (instead of having to specify with an environment variable). (#2593)

### Changed

- Upgraded all `semconv` package use to `v1.12.0`. (#2589)

## [1.8.0/0.33.0] - 2022-07-08

### Added

- The `go.opentelemetry.io/contrib/propagators/autoprop` package to provide configuration of propagators with useful defaults and envar support. (#2258)
- `WithPublicEndpointFn` hook to dynamically detect public HTTP requests and set their trace parent as a link. (#2342)

### Fixed

- Fix the `otelhttp`, `otelgin`, `otelmacaron`, `otelrestful` middlewares
  by using `SpanKindServer` when deciding the `SpanStatus`.
  This makes `4xx` response codes to not be an error anymore. (#2427)

## [1.7.0/0.32.0] - 2022-04-28

### Added

- Consistent probability sampler implementation. (#1379)

### Changed

- Upgraded all `semconv` package use to `v1.10.0`.
  This includes a backwards incompatible change for the `otelgocql` package to conform with the specification [change](https://github.com/open-telemetry/opentelemetry-specification/pull/1973).
  The `db.cassandra.keyspace` attribute is now transmitted as the `db.name` attribute. (#2222)

### Fixed

- Fix the `otelmux` middleware by using `SpanKindServer` when deciding the `SpanStatus`.
  This makes `4xx` response codes to not be an error anymore. (#1973)
- Fixed jaegerremote sampler not behaving properly with per operation strategy set. (#2137)
- Stopped injecting propagation context into response headers in otelhttp. (#2180)
- Fix issue where attributes for DynamoDB were not added because of a string miss match. (#2272)

### Removed

- Drop support for Go 1.16.
  The project currently only supports Go 1.17 and above. (#2314)

## [1.6.0/0.31.0] - 2022-03-28

### Added

- The project is now tested against Go 1.18 (in addition to the existing 1.16 and 1.17) (#1976)

### Changed

- Upgraded all dependencies on stable modules from `go.opentelemetry.io/otel` from v1.5.0 to v1.6.1. (#2134)
- Upgraded all dependencies on metric modules from `go.opentelemetry.io/otel` from v0.27.0 to v0.28.0. (#1977)

### Fixed

- otelhttp: Avoid panic by adding nil check to `wrappedBody.Close` (#2164)

## [1.5.0/0.30.0/0.1.0] - 2022-03-16

### Added

- Added the `go.opentelemetry.io/contrib/samplers/jaegerremote` package.
  This package implements the Jaeger remote sampler for OpenTelemetry Go. (#936)
- DynamoDB spans created with the `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` package now have the appropriate database attributes added for the operation being performed.
  These attributes are detected automatically, but it is also now possible to provide a custom function to set attributes using `WithAttributeSetter`. (#1582)
- Add resource detector for GCP cloud function. (#1584)
- Add OpenTracing baggage extraction to the OpenTracing propagator in `go.opentelemetry.io/contrib/propagators/ot`. (#1880)

### Fixed

- Fix the `echo` middleware by using `SpanKind.SERVER` when deciding the `SpanStatus`.
  This makes `4xx` response codes to not be an error anymore. (#1848)

### Removed

- The deprecated `go.opentelemetry.io/contrib/exporters/metric/datadog` module is removed. (#1920)
- The deprecated `go.opentelemetry.io/contrib/exporters/metric/dogstatsd` module is removed. (#1920)
- The deprecated `go.opentelemetry.io/contrib/exporters/metric/cortex` module is removed.
  Use the `go.opentelemetry.io/otel/exporters/otlp/otlpmetric` exporter as a replacement to send data to a collector which can then export with its PRW exporter. (#1920)

## [1.4.0/0.29.0] - 2022-02-14

### Added

- Add `WithClientTrace` option to `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`. (#875)

### Changed

- All metric instruments from the `go.opentelemetry.io/contrib/instrumentation/runtime` package have been renamed from `runtime.go.*` to `process.runtime.go.*` so as to comply with OpenTelemetry semantic conventions. (#1549)

### Fixed

- Change the `http-server-duration` instrument in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` to record milliseconds instead of microseconds.
  This changes fixes the code to comply with the OpenTelemetry specification. (#1414, #1537)
- Fixed the region reported by the `"go.opentelemetry.io/contrib/detectors/gcp".CloudRun` detector to comply with the OpenTelemetry specification.
  It no longer includes the project scoped region path, instead just the region. (#1546)
- The `"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp".Transport` type now correctly handles protocol switching responses.
  The returned response body implements the `io.ReadWriteCloser` interface if the underlying one does.
  This ensures that protocol switching requests receive a response body that they can write to. (#1329, #1628)

### Deprecated

- The `go.opentelemetry.io/contrib/exporters/metric/datadog` module is deprecated. (#1639)
- The `go.opentelemetry.io/contrib/exporters/metric/dogstatsd` module is deprecated. (#1639)
- The `go.opentelemetry.io/contrib/exporters/metric/cortex` module is deprecated.
  Use the go.opentelemetry.io/otel/exporters/otlp/otlpmetric exporter as a replacement to send data to a collector which can then export with its PRW exporter. (#1639)

### Removed

- Remove the `MinMaxSumCount` from cortex and datadog exporter. (#1554)
- The `go.opentelemetry.io/contrib/exporters/metric/dogstatsd` exporter no longer support exporting histogram or exact data points. (#1639)
- The `go.opentelemetry.io/contrib/exporters/metric/datadog` exporter no longer support exporting exact data points. (#1639)

## [1.3.0/0.28.0] - 2021-12-10

### ⚠️ Notice ⚠️

We have updated the project minimum supported Go version to 1.16

### Changed

- `otelhttptrace.NewClientTrace` now uses `TracerProvider` from the parent context if one exists and none was set with `WithTracerProvider` (#874)

### Fixed

- The `"go.opentelemetry.io/contrib/detector/aws/ecs".Detector` no longer errors if not running in ECS. (#1428)
- `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux`
  does not require instrumented HTTP handlers to call `Write` nor
  `WriteHeader` anymore. (#1443)

## [1.2.0/0.27.0] - 2021-11-15

### Changed

- Update dependency on the `go.opentelemetry.io/otel` project to `v1.2.0`.
- `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig`
  updated to ensure access to the `TracerProvider`.
  - A `NewTracerProvider()` function is available to construct a recommended
    `TracerProvider` configuration.
  - `AllRecommendedOptions()` has been renamed to `WithRecommendedOptions()`
    and takes a `TracerProvider` as an argument.
  - `EventToCarrier()` and `Propagator()` are now `WithEventToCarrier()` and
    `WithPropagator()` to reflect that they return `Option` implementations.

## [1.1.1/0.26.1] - 2021-11-04

### Changed

- The `Transport`, `Handler`, and HTTP client convenience wrappers in the `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` package now use the `TracerProvider` from the parent context if one exists and none was explicitly set when configuring the instrumentation. (#873)
- Semantic conventions now use `go.opentelemetry.io/otel/semconv/v1.7.0"`. (#1385)

## [1.1.0/0.26.0] - 2021-10-28

Update dependency on the `go.opentelemetry.io/otel` project to `v1.1.0`.

### Added

- Add instrumentation for the `github.com/aws/aws-lambda-go` package. (#983)
- Add resource detector for AWS Lambda. (#983)
- Add `WithTracerProvider` option for `otelhttptrace.NewClientTrace`. (#1128)
- Add optional AWS X-Ray configuration module for AWS Lambda Instrumentation. (#984)

### Fixed

- The `go.opentelemetry.io/contrib/propagators/ot` propagator returns the words `true` or `false` for the `ot-tracer-sampled` header instead of numerical `0` and `1`. (#1358)

## [1.0.0/0.25.0] - 2021-10-06

- Resource detectors and propagators (with the exception of `go.
  opentelemetry.io/contrib/propagators/opencensus`) are now stable and
  released at v1.0.0.
- Update dependency on the `go.opentelemetry.io/otel` project to `v1.0.1`.
- Update dependency on `go.opentelemetry.io/otel/metric` to `v0.24.0`.

## [0.24.0] - 2021-09-21

- Update dependency on the `go.opentelemetry.io/otel` project to `v1.0.0`.

## [0.23.0] - 2021-09-08

### Added

- Add `WithoutSubSpans`, `WithRedactedHeaders`, `WithoutHeaders`, and `WithInsecureHeaders` options for `otelhttptrace.NewClientTrace`. (#879)

### Changed

- Split `go.opentelemetry.io/contrib/propagators` module into `b3`, `jaeger`, `ot` modules. (#985)
- `otelmongodb` span attributes, name and span status now conform to specification. (#769)
- Migrated EC2 resource detector support from root module `go.opentelemetry.io/contrib/detectors/aws` to a separate EC2 resource detector module `go.opentelemetry.io/contrib/detectors/aws/ec2` (#1017)
- Add `cloud.provider` and `cloud.platform` to AWS detectors. (#1043)
- `otelhttptrace.NewClientTrace` now redacts known sensitive headers by default. (#879)

### Fixed

- Fix span not marked as error in `otelhttp.Transport` when `RoundTrip` fails with an error. (#950)

## [0.22.0] - 2021-07-26

### Added

- Add the `zpages` span processor. (#894)

### Changed

- The `b3.B3` type has been removed.
  `b3.New()` and `b3.WithInjectEncoding(encoding)` are added to replace it. (#868)

### Fixed

- Fix deadlocks and race conditions in `otelsarama.WrapAsyncProducer`.
  The `messaging.message_id` and `messaging.kafka.partition` attributes are now not set if a message was not processed. (#754) (#755) (#881)
- Fix `otelsarama.WrapAsyncProducer` so that the messages from the `Errors` channel contain the original `Metadata`. (#754)

## [0.21.0] - 2021-06-18

### Fixed

- Dockerfile based examples for `otelgin` and `otelmacaron`. (#767)

### Changed

- Supported minimum version of Go bumped from 1.14 to 1.15. (#787)
- EKS Resource Detector now use the Kubernetes Go client to obtain the ConfigMap. (#813)

### Removed

- Remove service name from `otelmongodb` configuration and span attributes. (#763)

## [0.20.0] - 2021-04-23

### Changed

- The `go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo` instrumentation now accepts a `WithCommandAttributeDisabled`,
   so the caller can specify whether to opt-out of tracing the mongo command. (#712)
- Upgrade to v0.20.0 of `go.opentelemetry.io/otel`. (#758)
- The B3 and Jaeger propagators now store their debug or deferred state in the context.Context instead of the SpanContext. (#758)

## [0.19.0] - 2021-03-19

### Changed

- Upgrade to v0.19.0 of `go.opentelemetry.io/otel`.
- Fix Span names created in HTTP Instrumentation package to conform with guidelines. (#757)

## [0.18.0] - 2021-03-04

### Fixed

- `otelmemcache` no longer sets span status to OK instead of leaving it unset. (#477)
- Fix goroutine leak in gRPC `StreamClientInterceptor`. (#581)

### Removed

- Remove service name from `otelmemcache` configuration and span attributes. (#477)

## [0.17.0] - 2021-02-15

### Added

- Add `ot-tracer` propagator (#562)

### Changed

- Rename project default branch from `master` to `main`.

### Fixed

- Added failure message for AWS ECS resource detector for better debugging (#568)
- Goroutine leak in gRPC StreamClientInterceptor while streamer returns an error. (#581)

## [0.16.0] - 2021-01-13

### Fixed

- Fix module path for AWS ECS resource detector (#517)

## [0.15.1] - 2020-12-14

### Added

- Add registry link check to `Makefile` and pre-release script. (#446)
- A new AWS X-Ray ID Generator (#459)
- Migrate CircleCI jobs to GitHub Actions (#476)
- Add CodeQL GitHub Action (#506)
- Add gosec workflow to GitHub Actions (#507)

### Fixed

- Fixes the body replacement in otelhttp to not to mutate a nil body. (#484)

## [0.15.0] - 2020-12-11

### Added

- A new Amazon EKS resource detector. (#465)
- A new `gcp.CloudRun` detector for detecting resource from a Cloud Run instance. (#455)

## [0.14.0] - 2020-11-20

### Added

- `otelhttp.{Get,Head,Post,PostForm}` convenience wrappers for their `http` counterparts. (#390)
- The AWS detector now adds the cloud zone, host image ID, host type, and host name to the returned `Resource`. (#410)
- Add Amazon ECS Resource Detector for AWS X-Ray. (#466)
- Add propagator for AWS X-Ray (#462)

### Changed

- Add semantic version to `Tracer` / `Meter` created by instrumentation packages `otelsaram`, `otelrestful`, `otelmongo`, `otelhttp` and `otelhttptrace`. (#412)
- Update instrumentation guidelines about tracer / meter semantic version. (#412)
- Replace internal tracer and meter helpers by helpers from `go.opentelemetry.io/otel`. (#414)
- gRPC instrumentation sets span attribute `rpc.grpc.status_code`. (#453)

## Fixed

- `/detectors/aws` no longer fails if instance metadata is not available (e.g. not running in AWS) (#401)
- The AWS detector now returns a partial resource and an appropriate error if it encounters an error part way through determining a `Resource` identity. (#410)
- The `host` instrumentation unit test has been updated to not depend on the system it runs on. (#426)

## [0.13.0] - 2020-10-09

## Added

- A Jaeger propagator. (#375)

## Changed

- The `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` package instrumentation no longer accepts a `Tracer` as an argument to the interceptor function.
   Instead, a new `WithTracerProvider` option is added to configure the `TracerProvider` used when creating the `Tracer` for the instrumentation. (#373)
- The `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron` instrumentation now accepts a `TracerProvider` rather than a `Tracer`. (#374)
- Remove `go.opentelemetry.io/otel/sdk` dependency from instrumentation. (#381)
- Use `httpsnoop` in `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` to ensure `http.ResponseWriter` additional interfaces are preserved. (#388)

### Fixed

- The `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho.Middleware` no longer sends duplicate errors to the global `ErrorHandler`. (#377, #364)
- The import comment in `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` is now correctly quoted. (#379)
- The B3 propagator sets the sample bitmask when the sampling decision is `debug`. (#369)

## [0.12.0] - 2020-09-25

### Added

- Benchmark tests for the gRPC instrumentation. (#296)
- Integration testing for the gRPC instrumentation. (#297)
- Allow custom labels to be added to net/http metrics. (#306)
- Added B3 propagator, moving it out of open.telemetry.io/otel repo. (#344)

### Changed

- Unify instrumentation about provider options for `go.mongodb.org/mongo-driver`, `gin-gonic/gin`, `gorilla/mux`,
  `labstack/echo`, `emicklei/go-restful`, `bradfitz/gomemcache`, `Shopify/sarama`, `net/http` and `beego`. (#303)
- Update instrumentation guidelines about uniform provider options. Also, update style guide. (#303)
- Make config struct of instrumentation unexported. (#303)
- Instrumentations have been updated to adhere to the [configuration style guide's](https://github.com/open-telemetry/opentelemetry-go/blob/master/CONTRIBUTING.md#config)
   updated recommendation to use `newConfig()` instead of `configure()`. (#336)
- A new instrumentation naming scheme is implemented to avoid package name conflicts for instrumented packages while still remaining discoverable. (#359)
  - `google.golang.org/grpc` -> `google.golang.org/grpc/otelgrpc`
  - `go.mongodb.org/mongo-driver` -> `go.mongodb.org/mongo-driver/mongo/otelmongo`
  - `net/http` -> `net/http/otelhttp`
  - `net/http/httptrace` -> `net/http/httptrace/otelhttptrace`
  - `github.com/labstack/echo` -> `github.com/labstack/echo/otelecho`
  - `github.com/bradfitz/gomemcache` -> `github.com/bradfitz/gomemcache/memcache/otelmemcache`
  - `github.com/gin-gonic/gin` -> `github.com/gin-gonic/gin/otelgin`
  - `github.com/gocql/gocql` -> `github.com/gocql/gocql/otelgocql`
  - `github.com/emicklei/go-restful` -> `github.com/emicklei/go-restful/otelrestful`
  - `github.com/Shopify/sarama` -> `github.com/Shopify/sarama/otelsarama`
  - `github.com/gorilla/mux` -> `github.com/gorilla/mux/otelmux`
  - `github.com/astaxie/beego` -> `github.com/astaxie/beego/otelbeego`
  - `gopkg.in/macaron.v1` -> `gopkg.in/macaron.v1/otelmacaron`
- Rename `OTelBeegoHandler` to `Handler` in the `go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego` package. (#359)
- Replace `WithTracer` with `WithTracerProvider` in the `go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron` instrumentation. (#374)

## [0.11.0] - 2020-08-25

### Added

- Top-level `Version()` and `SemVersion()` functions defining the current version of the contrib package. (#225)
- Instrumentation for the `github.com/astaxie/beego` package. (#200)
- Instrumentation for the `github.com/bradfitz/gomemcache` package. (#204)
- Host metrics instrumentation. (#231)
- Cortex histogram and distribution support. (#237)
- Cortex example project. (#238)
- Cortex HTTP authentication. (#246)

### Changed

- Remove service name as a parameter of Sarama instrumentation. (#221)
- Replace `WithTracer` with `WithTracerProvider` in Sarama instrumentation. (#221)
- Switch to use common top-level module `SemVersion()` when creating versioned tracer in `bradfitz/gomemcache`. (#226)
- Use `IntegrationShouldRun` in `gomemcache_test`. (#254)
- Use Go 1.15 for CI builds. (#236)
- Improved configuration for `runtime` instrumentation. (#224)

### Fixed

- Update dependabot configuration to include newly added `bradfitz/gomemcache` package. (#226)
- Correct `runtime` instrumentation name. (#241)

## [0.10.1] - 2020-08-13

### Added

- The `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc` module has been added to replace the instrumentation that had previoiusly existed in the `go.opentelemetry.io/otel/instrumentation/grpctrace` package. (#189)
- Instrumentation for the stdlib `net/http` and `net/http/httptrace` packages. (#190)
- Initial Cortex exporter. (#202, #205, #210, #211, #215)

### Fixed

- Bump google.golang.org/grpc from 1.30.0 to 1.31.0. (#166)
- Bump go.mongodb.org/mongo-driver from 1.3.5 to 1.4.0 in /instrumentation/go.mongodb.org/mongo-driver. (#170)
- Bump google.golang.org/grpc in /instrumentation/github.com/gin-gonic/gin. (#173)
- Bump google.golang.org/grpc in /instrumentation/github.com/labstack/echo. (#176)
- Bump google.golang.org/grpc from 1.30.0 to 1.31.0 in /instrumentation/github.com/Shopify/sarama. (#179)
- Bump cloud.google.com/go from 0.61.0 to 0.63.0 in /detectors/gcp. (#181, #199)
- Bump github.com/aws/aws-sdk-go from 1.33.15 to 1.34.1 in /detectors/aws. (#184, #192, #193, #198, #201, #203)
- Bump github.com/golangci/golangci-lint from 1.29.0 to 1.30.0 in /tools. (#186)
- Setup CI to run tests that require external resources (Cassandra and MongoDB). (#191)
- Bump github.com/Shopify/sarama from 1.26.4 to 1.27.0 in /instrumentation/github.com/Shopify/sarama. (#206)

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

[Unreleased]: https://github.com/open-telemetry/opentelemetry-go-contrib/compare/v1.34.0...HEAD
[1.34.0/0.59.0/0.28.0/0.14.0/0.9.0/0.7.0/0.6.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.34.0
[1.33.0/0.58.0/0.27.0/0.13.0/0.8.0/0.6.0/0.5.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.33.0
[1.32.0/0.57.0/0.26.0/0.12.0/0.7.0/0.5.0/0.4.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.32.0
[1.31.0/0.56.0/0.25.0/0.11.0/0.6.0/0.4.0/0.3.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.31.0
[1.30.0/0.55.0/0.24.0/0.10.0/0.5.0/0.3.0/0.2.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.30.0
[1.29.0/0.54.0/0.23.0/0.9.0/0.4.0/0.2.0/0.1.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.29.0
[1.28.0/0.53.0/0.22.0/0.8.0/0.3.0/0.1.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.28.0
[1.27.0/0.52.0/0.21.0/0.7.0/0.2.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.27.0
[1.26.0/0.51.0/0.20.0/0.6.0/0.1.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.26.0
[1.25.0/0.50.0/0.19.0/0.5.0/0.0.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.25.0
[1.24.0/0.49.0/0.18.0/0.4.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.24.0
[1.23.0/0.48.0/0.17.0/0.3.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.23.0
[1.22.0/0.47.0/0.16.0/0.2.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.22.0
[1.21.1/0.46.1/0.15.1/0.1.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.21.1
[1.21.0/0.46.0/0.15.0/0.1.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.21.0
[1.20.0/0.45.0/0.14.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.20.0
[1.19.0/0.44.0/0.13.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.19.0
[1.18.0/0.43.0/0.12.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.18.0
[1.17.0/0.42.0/0.11.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.17.0
[1.17.0-rc.1/0.42.0-rc.1/0.11.0-rc.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.17.0-rc.1
[1.16.1/0.41.1/0.10.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.16.1
[1.16.0/0.41.0/0.10.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.16.0
[1.16.0-rc.2/0.41.0-rc.2/0.10.0-rc.2]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.16.0-rc.2
[1.16.0-rc.1/0.41.0-rc.1/0.10.0-rc.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.16.0-rc.1
[1.15.0/0.40.0/0.9.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.15.0
[1.14.0/0.39.0/0.8.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.14.0
[1.13.0/0.38.0/0.7.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.13.0
[1.12.0/0.37.0/0.6.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.12.0
[1.11.1/0.36.4/0.5.2]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.11.1
[1.11.0/0.36.3/0.5.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.11.0
[0.36.2]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/zpages/v0.36.2
[0.36.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/zpages/v0.36.1
[0.36.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/zpages/v0.36.0
[1.10.0/0.35.0/0.5.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.10.0
[1.9.0/0.34.0/0.4.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.9.0
[1.8.0/0.33.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.8.0
[1.7.0/0.32.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.7.0
[1.6.0/0.31.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.6.0
[1.5.0/0.30.0/0.1.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.5.0
[1.4.0/0.29.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.4.0
[1.3.0/0.28.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.3.0
[1.2.0/0.27.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.2.0
[1.1.1/0.26.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.1.1
[1.1.0/0.26.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.1.0
[1.0.0/0.25.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v1.0.0
[0.24.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.24.0
[0.23.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.23.0
[0.22.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.22.0
[0.21.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.21.0
[0.20.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.20.0
[0.19.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.19.0
[0.18.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.18.0
[0.17.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.17.0
[0.16.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.16.0
[0.15.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.15.1
[0.15.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.15.0
[0.14.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.14.0
[0.13.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.13.0
[0.12.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.12.0
[0.11.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.11.0
[0.10.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.10.1
[0.10.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.10.0
[0.9.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.9.0
[0.8.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.8.0
[0.7.0]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.7.0
[0.6.1]: https://github.com/open-telemetry/opentelemetry-go-contrib/releases/tag/v0.6.1

<!-- Released section ended -->

[Go 1.24]: https://go.dev/doc/go1.24
[Go 1.23]: https://go.dev/doc/go1.23
[Go 1.22]: https://go.dev/doc/go1.22
[Go 1.21]: https://go.dev/doc/go1.21
[Go 1.20]: https://go.dev/doc/go1.20
[Go 1.19]: https://go.dev/doc/go1.19
[Go 1.18]: https://go.dev/doc/go1.18

[GO-2024-2687]: https://pkg.go.dev/vuln/GO-2024-2687

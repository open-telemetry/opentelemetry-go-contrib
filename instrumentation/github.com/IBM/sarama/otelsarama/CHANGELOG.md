# Changelog

All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This module adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- New module `go.opentelemetry.io/contrib/instrumentation/github.com/IBM/sarama/otelsarama` providing OpenTelemetry tracing instrumentation for the [github.com/IBM/sarama](https://github.com/IBM/sarama) Kafka client library. (#9367)
- `WrapSyncProducer` to instrument synchronous producers. (#9367)
- `WrapAsyncProducer` to instrument asynchronous producers. (#9367)
- `WrapConsumer` and `WrapPartitionConsumer` to instrument consumers. (#9367)
- `WrapConsumerGroupHandler` to instrument consumer group message processing. (#9367)
- `WithClient` option to enable background resolution of `messaging.kafka.cluster.id` with no blocking on the message path. (#9367)

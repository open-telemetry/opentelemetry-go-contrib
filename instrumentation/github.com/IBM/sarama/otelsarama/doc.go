// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelsarama provides OpenTelemetry instrumentation for the
// [github.com/IBM/sarama] Kafka client library.
//
// Use [WrapSyncProducer] or [WrapAsyncProducer] to instrument producers.
// Use [WrapConsumer] or [WrapPartitionConsumer] to instrument consumers.
// Use [WrapConsumerGroupHandler] to instrument consumer group handlers.
//
// Optionally pass [WithClient] to enable background cluster ID resolution,
// which adds the messaging.kafka.cluster.id attribute to all spans once
// the metadata fetch completes.
package otelsarama

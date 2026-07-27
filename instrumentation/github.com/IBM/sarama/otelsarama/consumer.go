// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"github.com/IBM/sarama"
)

// WrapPartitionConsumer wraps a sarama PartitionConsumer to add OpenTelemetry
// tracing. Each message received from Messages() will have a consumer span
// created and ended before being forwarded to the caller.
func WrapPartitionConsumer(pc sarama.PartitionConsumer, opts ...Option) sarama.PartitionConsumer {
	cfg := newConfig(opts...)
	wrapped := &partitionConsumer{
		PartitionConsumer: pc,
		dispatcher:        newConsumerMessagesDispatcherWrapper(pc, cfg),
	}
	go wrapped.dispatcher.Run()
	return wrapped
}

type partitionConsumer struct {
	sarama.PartitionConsumer
	dispatcher *consumerMessagesDispatcherWrapper
}

func (c *partitionConsumer) Messages() <-chan *sarama.ConsumerMessage {
	return c.dispatcher.Messages()
}

// WrapConsumer wraps a sarama Consumer to add OpenTelemetry tracing.
// Calls to ConsumePartition will return instrumented PartitionConsumers.
func WrapConsumer(c sarama.Consumer, opts ...Option) sarama.Consumer {
	cfg := newConfig(opts...)
	return &consumer{
		Consumer: c,
		cfg:      cfg,
	}
}

type consumer struct {
	sarama.Consumer
	cfg *config
}

func (c *consumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	pc, err := c.Consumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		return nil, err
	}
	wrapped := &partitionConsumer{
		PartitionConsumer: pc,
		dispatcher:        newConsumerMessagesDispatcherWrapper(pc, c.cfg),
	}
	go wrapped.dispatcher.Run()
	return wrapped, nil
}

// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sarama

import (
	"context"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/api/trace"
)

type partitionConsumer struct {
	sarama.PartitionConsumer
	messages chan *sarama.ConsumerMessage
}

// Messages returns the read channel for the messages that are returned by
// the broker.
func (pc *partitionConsumer) Messages() <-chan *sarama.ConsumerMessage {
	return pc.messages
}

// WrapPartitionConsumer wraps a sarama.PartitionConsumer causing each received
// message to be traced.
func WrapPartitionConsumer(serviceName string, pc sarama.PartitionConsumer, opts ...Option) sarama.PartitionConsumer {
	cfg := newConfig(serviceName, opts...)

	wrapped := &partitionConsumer{
		PartitionConsumer: pc,
		messages:          make(chan *sarama.ConsumerMessage),
	}
	go func() {
		msgs := pc.Messages()

		for msg := range msgs {
			// Extract a span context from message to link.
			carrier := NewConsumerMessageCarrier(msg)
			parentSpanContext := trace.RemoteSpanContextFromContext(propagation.ExtractHTTP(context.Background(), cfg.Propagators, carrier))

			// Create a span.
			attrs := []kv.KeyValue{
				standard.ServiceNameKey.String(cfg.ServiceName),
				standard.MessagingSystemKey.String("kafka"),
				standard.MessagingDestinationKindKeyTopic,
				standard.MessagingDestinationKey.String(msg.Topic),
				standard.MessagingOperationReceive,
				standard.MessagingMessageIDKey.Int64(msg.Offset),
				kafkaPartitionKey.Int32(msg.Partition),
			}
			opts := []trace.StartOption{
				trace.WithAttributes(attrs...),
				trace.WithSpanKind(trace.SpanKindConsumer),
			}
			if parentSpanContext.IsValid() {
				opts = append(opts, trace.LinkedTo(parentSpanContext))
			}
			newCtx, span := cfg.Tracer.Start(context.Background(), "kafka.consume", opts...)

			// Inject current span context, so consumers can use it to propagate span.
			propagation.InjectHTTP(newCtx, cfg.Propagators, carrier)

			// Send messages back to user.
			wrapped.messages <- msg

			span.End()
		}
		close(wrapped.messages)
	}()
	return wrapped
}

type consumer struct {
	sarama.Consumer

	serviceName string
	opts        []Option
}

// ConsumePartition invokes Consumer.ConsumePartition and wraps the resulting
// PartitionConsumer.
func (c *consumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	pc, err := c.Consumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		return nil, err
	}
	return WrapPartitionConsumer(c.serviceName, pc, c.opts...), nil
}

// WrapConsumer wraps a sarama.Consumer wrapping any PartitionConsumer created
// via Consumer.ConsumePartition.
func WrapConsumer(serviceName string, c sarama.Consumer, opts ...Option) sarama.Consumer {
	return &consumer{
		Consumer:    c,
		serviceName: serviceName,
		opts:        opts,
	}
}

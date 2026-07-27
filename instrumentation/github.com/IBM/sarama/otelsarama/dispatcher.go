// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// consumerMessagesDispatcher is the interface for types that expose a Messages channel.
type consumerMessagesDispatcher interface {
	Messages() <-chan *sarama.ConsumerMessage
}

// consumerMessagesDispatcherWrapper wraps a dispatcher and processes messages
// by creating a consumer span for each one, then forwarding to the output channel.
type consumerMessagesDispatcherWrapper struct {
	d        consumerMessagesDispatcher
	messages chan *sarama.ConsumerMessage
	cfg      *config
}

func newConsumerMessagesDispatcherWrapper(d consumerMessagesDispatcher, cfg *config) *consumerMessagesDispatcherWrapper {
	return &consumerMessagesDispatcherWrapper{
		d:        d,
		messages: make(chan *sarama.ConsumerMessage),
		cfg:      cfg,
	}
}

// Messages returns the channel of instrumented consumer messages.
func (w *consumerMessagesDispatcherWrapper) Messages() <-chan *sarama.ConsumerMessage {
	return w.messages
}

// Run reads messages from the underlying dispatcher, creates consumer spans,
// injects the span context back into the message for downstream propagation,
// and forwards the message. It exits when the upstream Messages channel is closed.
func (w *consumerMessagesDispatcherWrapper) Run() {
	defer close(w.messages)
	for msg := range w.d.Messages() {
		ctx := w.cfg.Propagators.Extract(context.Background(), NewConsumerMessageCarrier(msg))

		attrs := []attribute.KeyValue{
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationNameKey.String(msg.Topic),
			semconv.MessagingDestinationPartitionIDKey.String(fmt.Sprintf("%d", msg.Partition)),
			semconv.MessagingKafkaOffsetKey.Int64(msg.Offset),
			semconv.MessagingOperationNameKey.String("process"),
			semconv.MessagingOperationTypeKey.String("process"),
		}

		if clusterID := w.cfg.clusterID.Load(); clusterID != nil {
			attrs = append(attrs, attribute.String("messaging.kafka.cluster.id", *clusterID))
		}

		opts := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(attrs...),
		}

		newCtx, span := w.cfg.Tracer.Start(ctx, fmt.Sprintf("process %s", msg.Topic), opts...)

		// Re-inject updated context so downstream code can link to this span.
		w.cfg.Propagators.Inject(newCtx, newConsumerMessageTextMapCarrierAdapter(msg))

		w.messages <- msg
		span.End()
	}
}

// consumerMessageTextMapCarrierAdapter adapts *sarama.ConsumerMessage for injection
// by allowing Set to append headers (consumer messages are read-only in practice,
// but we need to propagate context forward through the message).
type consumerMessageTextMapCarrierAdapter struct {
	msg *sarama.ConsumerMessage
}

func newConsumerMessageTextMapCarrierAdapter(msg *sarama.ConsumerMessage) propagation.TextMapCarrier {
	return &consumerMessageTextMapCarrierAdapter{msg: msg}
}

func (c *consumerMessageTextMapCarrierAdapter) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h != nil && string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *consumerMessageTextMapCarrierAdapter) Set(key, val string) {
	for i, h := range c.msg.Headers {
		if h != nil && string(h.Key) == key {
			c.msg.Headers[i].Value = []byte(val)
			return
		}
	}
	c.msg.Headers = append(c.msg.Headers, &sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

func (c *consumerMessageTextMapCarrierAdapter) Keys() []string {
	keys := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		if h != nil {
			keys = append(keys, string(h.Key))
		}
	}
	return keys
}

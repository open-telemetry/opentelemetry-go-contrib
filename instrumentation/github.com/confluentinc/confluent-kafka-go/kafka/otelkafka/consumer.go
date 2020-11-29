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

package otelkafka

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

// NewConsumer calls kafka.NewConsumer and wraps the resulting Consumer.
func NewConsumer(conf *kafka.ConfigMap, opts ...Option) (*Consumer, error) {
	c, err := kafka.NewConsumer(conf)
	if err != nil {
		return nil, err
	}
	return WrapConsumer(c, opts...), nil
}

// A Consumer wraps a kafka.Consumer.
type Consumer struct {
	*kafka.Consumer
	cfg    *config
	events chan kafka.Event
	prev   oteltrace.Span
}

// WrapConsumer wraps a kafka.Consumer so that any consumed events are traced.
func WrapConsumer(c *kafka.Consumer, opts ...Option) *Consumer {
	wrapped := &Consumer{
		Consumer: c,
		cfg:      newConfig(opts...),
	}
	wrapped.events = wrapped.traceEventsChannel(c.Events())
	return wrapped
}

func (c *Consumer) traceEventsChannel(in chan kafka.Event) chan kafka.Event {
	// in will be nil when consuming via the events channel is not enabled
	if in == nil {
		return nil
	}

	out := make(chan kafka.Event, 1)
	go func() {
		defer close(out)
		for evt := range in {
			var next oteltrace.Span

			// only trace messages
			if msg, ok := evt.(*kafka.Message); ok {
				next = c.startSpan(msg)
			}

			out <- evt

			if c.prev != nil {
				c.prev.End()
			}
			c.prev = next
		}
		// finish any remaining span
		if c.prev != nil {
			c.prev.End()
			c.prev = nil
		}
	}()

	return out
}

func (c *Consumer) startSpan(msg *kafka.Message) oteltrace.Span {
	// Extract a span context from message.
	carrier := NewMessageCarrier(msg)
	parentSpanContext := c.cfg.Propagators.Extract(c.cfg.ctx, carrier)

	opts := []oteltrace.SpanOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
		oteltrace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingOperationReceive,
			semconv.MessagingDestinationKindKeyTopic,
			semconv.MessagingDestinationKey.String(*msg.TopicPartition.Topic),
			label.Key(kafkaMessageKeyField).String(string(msg.Key)),
			label.Key(kafkaPartitionField).Int32(msg.TopicPartition.Partition)),
	}

	// Start a span using parentSpanContext
	newCtx, span := c.cfg.Tracer.Start(parentSpanContext, fmt.Sprintf("%s receive", *msg.TopicPartition.Topic), opts...)

	// Inject current span context, so consumers can use it to propagate span for furthur processing
	c.cfg.Propagators.Inject(newCtx, carrier)
	return span
}

// Close calls the underlying Consumer.Close and if polling is enabled, finishes
// any remaining span.
func (c *Consumer) Close() error {
	err := c.Consumer.Close()
	// we only close the previous span if consuming via the events channel is
	// not enabled, because otherwise there would be a data race from the
	// consuming goroutine.
	if c.events == nil && c.prev != nil {
		c.prev.End()
		c.prev = nil
	}
	return err
}

// Events returns the kafka Events channel (if enabled). Message events will be
// traced.
func (c *Consumer) Events() chan kafka.Event {
	return c.events
}

// Poll polls the consumer for messages or events. Message events will be
// traced.
func (c *Consumer) Poll(timeoutMS int) (event kafka.Event) {
	if c.prev != nil {
		c.prev.End()
		c.prev = nil
	}
	evt := c.Consumer.Poll(timeoutMS)
	if msg, ok := evt.(*kafka.Message); ok {
		c.prev = c.startSpan(msg)
	}
	return evt
}

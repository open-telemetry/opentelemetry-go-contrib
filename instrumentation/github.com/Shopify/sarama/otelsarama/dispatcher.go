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

package otelsarama // import "go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"

import (
	"context"
	"strconv"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

type consumerMessagesDispatcher interface {
	Messages() <-chan *sarama.ConsumerMessage
}

type consumerMessagesDispatcherWrapper struct {
	d        consumerMessagesDispatcher
	messages chan *sarama.ConsumerMessage

	cfg config
}

func newConsumerMessagesDispatcherWrapper(d consumerMessagesDispatcher, cfg config) *consumerMessagesDispatcherWrapper {
	return &consumerMessagesDispatcherWrapper{
		d:        d,
		messages: make(chan *sarama.ConsumerMessage),
		cfg:      cfg,
	}
}

// Messages returns the read channel for the messages that are returned by
// the broker.
func (w *consumerMessagesDispatcherWrapper) Messages() <-chan *sarama.ConsumerMessage {
	return w.messages
}

func (w *consumerMessagesDispatcherWrapper) Run() {
	msgs := w.d.Messages()

	for msg := range msgs {
		// Extract a span context from message to link.
		carrier := NewConsumerMessageCarrier(msg)
		parentSpanContext := w.cfg.Propagators.Extract(context.Background(), carrier)

		// Create a span.
		attrs := []attribute.KeyValue{
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationKindTopic,
			semconv.MessagingDestinationKey.String(msg.Topic),
			semconv.MessagingOperationReceive,
			semconv.MessagingMessageIDKey.String(strconv.FormatInt(msg.Offset, 10)),
			semconv.MessagingKafkaPartitionKey.Int64(int64(msg.Partition)),
		}
		opts := []trace.SpanStartOption{
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindConsumer),
		}
		newCtx, span := w.cfg.Tracer.Start(parentSpanContext, "kafka.consume", opts...)

		// Inject current span context, so consumers can use it to propagate span.
		w.cfg.Propagators.Inject(newCtx, carrier)

		// Send messages back to user.
		w.messages <- msg

		span.End()
	}
	close(w.messages)
}

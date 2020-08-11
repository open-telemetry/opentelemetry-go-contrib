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
	"strconv"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/api/trace"
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
		parentSpanContext := propagation.ExtractHTTP(context.Background(), w.cfg.Propagators, carrier)

		// Create a span.
		attrs := []kv.KeyValue{
			standard.ServiceNameKey.String(w.cfg.ServiceName),
			standard.MessagingSystemKey.String("kafka"),
			standard.MessagingDestinationKindKeyTopic,
			standard.MessagingDestinationKey.String(msg.Topic),
			standard.MessagingOperationReceive,
			standard.MessagingMessageIDKey.String(strconv.FormatInt(msg.Offset, 10)),
			kafkaPartitionKey.Int32(msg.Partition),
		}
		opts := []trace.StartOption{
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindConsumer),
		}
		newCtx, span := w.cfg.Tracer.Start(parentSpanContext, "kafka.consume", opts...)

		// Inject current span context, so consumers can use it to propagate span.
		propagation.InjectHTTP(newCtx, w.cfg.Propagators, carrier)

		// Send messages back to user.
		w.messages <- msg

		span.End()
	}
	close(w.messages)
}

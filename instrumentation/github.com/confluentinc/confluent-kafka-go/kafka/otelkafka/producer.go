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

// Package kafka provides functions to trace the confluentinc/confluent-kafka-go package (https://github.com/confluentinc/confluent-kafka-go).

package otelkafka

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

// Producer wraps a kafka.Producer.
type Producer struct {
	*kafka.Producer
	cfg            *config
	produceChannel chan *kafka.Message
}

// NewProducer calls kafka.NewProducer and wraps the resulting Producer.
func NewProducer(conf *kafka.ConfigMap, opts ...Option) (*Producer, error) {
	p, err := kafka.NewProducer(conf)
	if err != nil {
		return nil, err
	}
	return WrapProducer(p, opts...), nil
}

// WrapProducer wraps a kafka.Producer so requests are traced.
func WrapProducer(p *kafka.Producer, opts ...Option) *Producer {
	wrapped := &Producer{
		Producer: p,
		cfg:      newConfig(opts...),
	}
	wrapped.produceChannel = wrapped.traceProduceChannel(p.ProduceChannel())
	return wrapped
}

func (p *Producer) traceProduceChannel(out chan *kafka.Message) chan *kafka.Message {
	if out == nil {
		return out
	}

	in := make(chan *kafka.Message, 1)
	go func() {
		for msg := range in {
			span := p.startSpan(msg)
			out <- msg
			span.End()
		}
	}()

	return in
}

func (p *Producer) startSpan(msg *kafka.Message) oteltrace.Span {
	opts := []oteltrace.SpanOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindProducer),
		oteltrace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationKindKeyTopic,
			semconv.MessagingDestinationKey.String(*msg.TopicPartition.Topic),
			label.Key(kafkaMessageKeyField).String(string(msg.Key)),
			label.Key(kafkaPartitionField).Int32(msg.TopicPartition.Partition),
		),
	}

	// If there's a span context in the message, use that as the parent context.
	carrier := NewMessageCarrier(msg)
	ctx := p.cfg.Propagators.Extract(p.cfg.ctx, carrier)
	ctx, span := p.cfg.Tracer.Start(ctx, fmt.Sprintf("%s send", *msg.TopicPartition.Topic), opts...)

	// Inject the span context so consumers can pick it up
	carrier = NewMessageCarrier(msg)
	p.cfg.Propagators.Inject(ctx, carrier)
	return span
}

// Close calls the underlying Producer.Close and also closes the internal
// wrapping producer channel.
func (p *Producer) Close() {
	close(p.produceChannel)
	p.Producer.Close()
}

// Produce calls the underlying Producer.Produce and traces the request.
func (p *Producer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	span := p.startSpan(msg)

	// If the user has selected a delivery channel, we will wrap it and
	// wait for the delivery event to finish the span
	if deliveryChan != nil {
		oldDeliveryChan := deliveryChan
		deliveryChan = make(chan kafka.Event)
		go func() {
			var err error
			evt := <-deliveryChan
			if msg, ok := evt.(*kafka.Message); ok {
				// delivery errors are returned via TopicPartition.Error
				err = msg.TopicPartition.Error
			}
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
			}
			span.End()
			oldDeliveryChan <- evt
		}()
	}

	err := p.Producer.Produce(msg, deliveryChan)
	// with no delivery channel, finish immediately
	if deliveryChan == nil {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}

	return err
}

// ProduceChannel returns a channel which can receive kafka Messages and will
// send them to the underlying producer channel.
func (p *Producer) ProduceChannel() chan *kafka.Message {
	return p.produceChannel
}

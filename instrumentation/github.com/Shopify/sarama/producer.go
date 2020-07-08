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
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/api/trace"
	"google.golang.org/grpc/codes"
)

type syncProducer struct {
	sarama.SyncProducer
	cfg config
}

// SendMessage calls sarama.SyncProducer.SendMessage and traces the request.
func (p *syncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	span := startProducerSpan(p.cfg, msg)
	partition, offset, err = p.SyncProducer.SendMessage(msg)
	finishProducerSpan(span, partition, offset, err)
	return partition, offset, err
}

// SendMessages calls sarama.SyncProducer.SendMessages and traces the requests.
func (p *syncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	// although there's only one call made to the SyncProducer, the messages are
	// treated individually, so we create a span for each one
	spans := make([]trace.Span, len(msgs))
	for i, msg := range msgs {
		spans[i] = startProducerSpan(p.cfg, msg)
	}
	err := p.SyncProducer.SendMessages(msgs)
	for i, span := range spans {
		finishProducerSpan(span, msgs[i].Partition, msgs[i].Offset, err)
	}
	return err
}

// WrapSyncProducer wraps a sarama.SyncProducer so that all produced messages
// are traced.
func WrapSyncProducer(serviceName string, producer sarama.SyncProducer, opts ...Option) sarama.SyncProducer {
	cfg := newConfig(serviceName, opts...)
	return &syncProducer{
		SyncProducer: producer,
		cfg:          cfg,
	}
}

type closeType int

const (
	closeSync  closeType = iota
	closeAsync closeType = iota
)

type asyncProducer struct {
	sarama.AsyncProducer
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
	close     chan closeType
	closeErr  chan error
}

// Input returns the input channel.
func (p *asyncProducer) Input() chan<- *sarama.ProducerMessage {
	return p.input
}

// Successes returns the successes channel.
func (p *asyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return p.successes
}

// Errors returns the errors channel.
func (p *asyncProducer) Errors() <-chan *sarama.ProducerError {
	return p.errors
}

// AsyncClose async close producer.
func (p *asyncProducer) AsyncClose() {
	p.close <- closeAsync
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed.
func (p *asyncProducer) Close() error {
	p.close <- closeSync
	return <-p.closeErr
}

// WrapAsyncProducer wraps a sarama.AsyncProducer so that all produced messages
// are traced. It requires the underlying sarama Config so we can know whether
// or not sucesses will be returned.
//
// If `Return.Successes` is false, there is no way to know partition and offset of
// the message.
func WrapAsyncProducer(serviceName string, saramaConfig *sarama.Config, p sarama.AsyncProducer, opts ...Option) sarama.AsyncProducer {
	cfg := newConfig(serviceName, opts...)
	if saramaConfig == nil {
		saramaConfig = sarama.NewConfig()
	}

	wrapped := &asyncProducer{
		AsyncProducer: p,
		input:         make(chan *sarama.ProducerMessage),
		successes:     make(chan *sarama.ProducerMessage),
		errors:        make(chan *sarama.ProducerError),
		close:         make(chan closeType),
		closeErr:      make(chan error),
	}
	go func() {
		spans := make(map[interface{}]trace.Span)
		defer close(wrapped.successes)
		defer close(wrapped.errors)
		for {
			select {
			case t := <-wrapped.close:
				switch t {
				case closeSync:
					wrapped.closeErr <- p.Close()
				case closeAsync:
					p.AsyncClose()
				}
			case msg := <-wrapped.input:
				msg.Metadata = uuid.New()
				span := startProducerSpan(cfg, msg)
				p.Input() <- msg
				if saramaConfig.Producer.Return.Successes {
					spans[msg.Metadata] = span
				} else {
					// If returning successes isn't enabled, we just finish the
					// span right away because there's no way to know when it will
					// be done.
					finishProducerSpan(span, msg.Partition, msg.Offset, nil)
				}
			case msg, ok := <-p.Successes():
				if !ok {
					// producer was closed, so exit
					return
				}
				key := msg.Metadata
				if span, ok := spans[key]; ok {
					delete(spans, key)
					finishProducerSpan(span, msg.Partition, msg.Offset, nil)
				}
				wrapped.successes <- msg
			case err, ok := <-p.Errors():
				if !ok {
					// producer was closed
					return
				}
				key := err.Msg.Metadata
				if span, ok := spans[key]; ok {
					delete(spans, key)
					finishProducerSpan(span, err.Msg.Partition, err.Msg.Offset, err.Err)
				}
				wrapped.errors <- err
			}
		}
	}()
	return wrapped
}

func startProducerSpan(cfg config, msg *sarama.ProducerMessage) trace.Span {
	// If there's a span context in the message, use that as the parent context.
	carrier := NewProducerMessageCarrier(msg)
	ctx := propagation.ExtractHTTP(context.Background(), cfg.Propagators, carrier)

	// Create a span.
	attrs := []kv.KeyValue{
		standard.ServiceNameKey.String(cfg.ServiceName),
		standard.MessagingSystemKey.String("kafka"),
		standard.MessagingDestinationKindKeyTopic,
		standard.MessagingDestinationKey.String(msg.Topic),
	}
	opts := []trace.StartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}
	ctx, span := cfg.Tracer.Start(ctx, "kafka.produce", opts...)

	// Inject current span context, so consumers can use it to propagate span.
	propagation.InjectHTTP(ctx, cfg.Propagators, carrier)

	return span
}

func finishProducerSpan(span trace.Span, partition int32, offset int64, err error) {
	span.SetAttributes(
		standard.MessagingMessageIDKey.Int64(offset),
		kafkaPartitionKey.Int32(partition),
	)
	if err != nil {
		span.SetStatus(codes.Internal, err.Error())
	}
	span.End()
}

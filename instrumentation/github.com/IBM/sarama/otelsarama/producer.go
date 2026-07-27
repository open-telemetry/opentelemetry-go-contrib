// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"context"
	"strconv"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// WrapSyncProducer wraps a sarama SyncProducer to add OpenTelemetry tracing
// to each produced message.
func WrapSyncProducer(saramaConfig *sarama.Config, producer sarama.SyncProducer, opts ...Option) sarama.SyncProducer {
	cfg := newConfig(opts...)
	return &syncProducer{
		SyncProducer: producer,
		cfg:          cfg,
		saramaConfig: saramaConfig,
	}
}

type syncProducer struct {
	sarama.SyncProducer
	cfg          *config
	saramaConfig *sarama.Config
}

func (p *syncProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	span := startProducerSpan(p.cfg, p.saramaConfig.Version, msg)
	partition, offset, err := p.SyncProducer.SendMessage(msg)
	finishProducerSpan(span, partition, offset, err)
	return partition, offset, err
}

func (p *syncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	spans := make([]trace.Span, len(msgs))
	for i, msg := range msgs {
		spans[i] = startProducerSpan(p.cfg, p.saramaConfig.Version, msg)
	}
	err := p.SyncProducer.SendMessages(msgs)
	for i, span := range spans {
		finishProducerSpan(span, msgs[i].Partition, msgs[i].Offset, err)
	}
	return err
}

// WrapAsyncProducer wraps a sarama AsyncProducer to add OpenTelemetry tracing
// to each produced message. The wrapped producer mirrors span state through
// the Successes and Errors channels.
//
// If saramaConfig.Producer.Return.Successes is false, spans are ended when
// the message is passed to the underlying producer. When Return.Successes is
// true, spans are ended when the corresponding success or error is received.
func WrapAsyncProducer(saramaConfig *sarama.Config, p sarama.AsyncProducer, opts ...Option) sarama.AsyncProducer {
	cfg := newConfig(opts...)
	wrapped := &asyncProducer{
		AsyncProducer: p,
		cfg:           cfg,
		input:         make(chan *sarama.ProducerMessage),
		successes:     make(chan *sarama.ProducerMessage),
		errors:        make(chan *sarama.ProducerError),
	}
	go wrapped.runInput(saramaConfig)
	if saramaConfig.Producer.Return.Successes {
		go wrapped.runSuccesses()
	}
	if saramaConfig.Producer.Return.Errors {
		go wrapped.runErrors()
	}
	return wrapped
}

type asyncProducerSpanKey struct{}

type asyncProducer struct {
	sarama.AsyncProducer
	cfg       *config
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
}

func (p *asyncProducer) Input() chan<- *sarama.ProducerMessage {
	return p.input
}

func (p *asyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return p.successes
}

func (p *asyncProducer) Errors() <-chan *sarama.ProducerError {
	return p.errors
}

func (p *asyncProducer) runInput(saramaConfig *sarama.Config) {
	for msg := range p.input {
		span := startProducerSpan(p.cfg, saramaConfig.Version, msg)
		if saramaConfig.Producer.Return.Successes {
			// stash span in msg metadata for retrieval on success/error
			msg.Metadata = withSpan(msg.Metadata, span)
		} else {
			// fire-and-forget: end span immediately
			span.End()
		}
		p.AsyncProducer.Input() <- msg
	}
}

func (p *asyncProducer) runSuccesses() {
	for msg := range p.AsyncProducer.Successes() {
		if span := spanFromMetadata(msg.Metadata); span != nil {
			finishProducerSpan(span, msg.Partition, msg.Offset, nil)
			msg.Metadata = withoutSpan(msg.Metadata)
		}
		p.successes <- msg
	}
	close(p.successes)
}

func (p *asyncProducer) runErrors() {
	for pe := range p.AsyncProducer.Errors() {
		if span := spanFromMetadata(pe.Msg.Metadata); span != nil {
			finishProducerSpan(span, pe.Msg.Partition, pe.Msg.Offset, pe.Err)
			pe.Msg.Metadata = withoutSpan(pe.Msg.Metadata)
		}
		p.errors <- pe
	}
	close(p.errors)
}

// spanMetadata wraps the user's original metadata together with the span.
type spanMetadata struct {
	original interface{}
	span     trace.Span
}

func withSpan(original interface{}, span trace.Span) interface{} {
	return &spanMetadata{original: original, span: span}
}

func withoutSpan(meta interface{}) interface{} {
	if sm, ok := meta.(*spanMetadata); ok {
		return sm.original
	}
	return meta
}

func spanFromMetadata(meta interface{}) trace.Span {
	if sm, ok := meta.(*spanMetadata); ok {
		return sm.span
	}
	return nil
}

func startProducerSpan(cfg *config, version sarama.KafkaVersion, msg *sarama.ProducerMessage) trace.Span {
	carrier := NewProducerMessageCarrier(msg)
	ctx := cfg.Propagators.Extract(context.Background(), carrier)

	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String("kafka"),
		semconv.MessagingDestinationNameKey.String(msg.Topic),
		semconv.MessagingOperationNameKey.String("send"),
		semconv.MessagingOperationTypeKey.String("send"),
	}

	if clusterID := cfg.clusterID.Load(); clusterID != nil {
		// messaging.kafka.cluster.id — targeted for semconv v1.44
		attrs = append(attrs, attribute.String("messaging.kafka.cluster.id", *clusterID))
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attrs...),
	}

	ctx, span := cfg.Tracer.Start(ctx, "send "+msg.Topic, opts...)

	// Inject trace context into message headers if Kafka version supports it.
	if version.IsAtLeast(sarama.V0_11_0_0) {
		cfg.Propagators.Inject(ctx, carrier)
	}

	return span
}

func finishProducerSpan(span trace.Span, partition int32, offset int64, err error) {
	span.SetAttributes(
		semconv.MessagingDestinationPartitionIDKey.String(strconv.Itoa(int(partition))),
		semconv.MessagingKafkaOffsetKey.Int64(offset),
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

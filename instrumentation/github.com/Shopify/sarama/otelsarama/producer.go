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
	"sync"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

type syncProducer struct {
	sarama.SyncProducer
	cfg          config
	saramaConfig *sarama.Config
}

// SendMessage calls sarama.SyncProducer.SendMessage and traces the request.
func (p *syncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	span := startProducerSpan(p.cfg, p.saramaConfig.Version, msg)
	partition, offset, err = p.SyncProducer.SendMessage(msg)
	finishProducerSpan(span, partition, offset, err)
	return partition, offset, err
}

// SendMessages calls sarama.SyncProducer.SendMessages and traces the requests.
func (p *syncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	// Although there's only one call made to the SyncProducer, the messages are
	// treated individually, so we create a span for each one
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

// WrapSyncProducer wraps a sarama.SyncProducer so that all produced messages
// are traced.
func WrapSyncProducer(saramaConfig *sarama.Config, producer sarama.SyncProducer, opts ...Option) sarama.SyncProducer {
	cfg := newConfig(opts...)
	if saramaConfig == nil {
		saramaConfig = sarama.NewConfig()
	}

	return &syncProducer{
		SyncProducer: producer,
		cfg:          cfg,
		saramaConfig: saramaConfig,
	}
}

type asyncProducer struct {
	sarama.AsyncProducer
	input         chan *sarama.ProducerMessage
	successes     chan *sarama.ProducerMessage
	errors        chan *sarama.ProducerError
	closeErr      chan error
	closeSig      chan struct{}
	closeAsyncSig chan struct{}
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
	close(p.input)
	close(p.closeAsyncSig)
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed.
//
// Due to the implement of sarama, some messages may lose successes or errors status
// while closing.
func (p *asyncProducer) Close() error {
	close(p.input)
	close(p.closeSig)
	return <-p.closeErr
}

type producerMessageContext struct {
	span           trace.Span
	metadataBackup interface{}
}

// WrapAsyncProducer wraps a sarama.AsyncProducer so that all produced messages
// are traced. It requires the underlying sarama Config so we can know whether
// or not successes will be returned.
//
// If `Return.Successes` is false, there is no way to know partition and offset of
// the message.
func WrapAsyncProducer(saramaConfig *sarama.Config, p sarama.AsyncProducer, opts ...Option) sarama.AsyncProducer {
	cfg := newConfig(opts...)
	if saramaConfig == nil {
		saramaConfig = sarama.NewConfig()
	}

	wrapped := &asyncProducer{
		AsyncProducer: p,
		input:         make(chan *sarama.ProducerMessage),
		successes:     make(chan *sarama.ProducerMessage),
		errors:        make(chan *sarama.ProducerError),
		closeErr:      make(chan error),
		closeSig:      make(chan struct{}),
		closeAsyncSig: make(chan struct{}),
	}

	var (
		mtx                     sync.Mutex
		producerMessageContexts = make(map[interface{}]producerMessageContext)
	)

	// Spawn Input producer goroutine.
	go func() {
		for {
			select {
			case <-wrapped.closeSig:
				wrapped.closeErr <- p.Close()
				return
			case <-wrapped.closeAsyncSig:
				p.AsyncClose()
				return
			case msg, ok := <-wrapped.input:
				if !ok {
					continue // wait for closeAsyncSig
				}
				span := startProducerSpan(cfg, saramaConfig.Version, msg)

				// Create message context, backend message metadata
				mc := producerMessageContext{
					metadataBackup: msg.Metadata,
					span:           span,
				}

				// Remember metadata using span ID as a cache key
				msg.Metadata = span.SpanContext().SpanID()
				if saramaConfig.Producer.Return.Successes {
					mtx.Lock()
					producerMessageContexts[msg.Metadata] = mc
					mtx.Unlock()
				} else {
					// If returning successes isn't enabled, we just finish the
					// span right away because there's no way to know when it will
					// be done.
					mc.span.End()
				}

				p.Input() <- msg
			}
		}
	}()

	// Sarama will consume all the successes and errors by itself while closing,
	// so we need to end these these spans ourseleves.
	var cleanupWg sync.WaitGroup

	// Spawn Successes consumer goroutine.
	cleanupWg.Add(1)
	go func() {
		defer func() {
			close(wrapped.successes)
			cleanupWg.Done()
		}()
		for msg := range p.Successes() {
			key := msg.Metadata
			mtx.Lock()
			if mc, ok := producerMessageContexts[key]; ok {
				delete(producerMessageContexts, key)
				finishProducerSpan(mc.span, msg.Partition, msg.Offset, nil)
				msg.Metadata = mc.metadataBackup // Restore message metadata
			}
			mtx.Unlock()
			wrapped.successes <- msg
		}
	}()

	// Spawn Errors consumer goroutine.
	cleanupWg.Add(1)
	go func() {
		defer func() {
			close(wrapped.errors)
			cleanupWg.Done()
		}()
		for errMsg := range p.Errors() {
			key := errMsg.Msg.Metadata
			mtx.Lock()
			if mc, ok := producerMessageContexts[key]; ok {
				delete(producerMessageContexts, key)
				finishProducerSpan(mc.span, errMsg.Msg.Partition, errMsg.Msg.Offset, errMsg.Err)
				errMsg.Msg.Metadata = mc.metadataBackup // Restore message metadata
			}
			mtx.Unlock()
			wrapped.errors <- errMsg
		}
	}()

	// Spawn spans cleanup goroutine.
	go func() {
		// wait until both consumer goroutines are closed
		cleanupWg.Wait()
		// end all remaining spans
		mtx.Lock()
		for _, mc := range producerMessageContexts {
			mc.span.End()
		}
		mtx.Unlock()
	}()

	return wrapped
}

func startProducerSpan(cfg config, version sarama.KafkaVersion, msg *sarama.ProducerMessage) trace.Span {
	// If there's a span context in the message, use that as the parent context.
	carrier := NewProducerMessageCarrier(msg)
	ctx := cfg.Propagators.Extract(context.Background(), carrier)

	// Create a span.
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String("kafka"),
		semconv.MessagingDestinationKindTopic,
		semconv.MessagingDestinationKey.String(msg.Topic),
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}
	ctx, span := cfg.Tracer.Start(ctx, "kafka.produce", opts...)

	if version.IsAtLeast(sarama.V0_11_0_0) {
		// Inject current span context, so consumers can use it to propagate span.
		cfg.Propagators.Inject(ctx, carrier)
	}

	return span
}

func finishProducerSpan(span trace.Span, partition int32, offset int64, err error) {
	span.SetAttributes(
		semconv.MessagingMessageIDKey.String(strconv.FormatInt(offset, 10)),
		semconv.MessagingKafkaPartitionKey.Int64(int64(partition)),
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

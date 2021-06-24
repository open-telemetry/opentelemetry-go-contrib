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

package otelsarama

import (
	"context"
	"errors"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestWrapSyncProducer(t *testing.T) {
	propagators := propagation.TraceContext{}
	var err error

	// Mock provider
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(t, cfg)

	// Wrap sync producer
	syncProducer := WrapSyncProducer(cfg, mockSyncProducer, WithTracerProvider(provider), WithPropagators(propagators))

	// Create message with span context
	ctx, _ := provider.Tracer(defaultTracerName).Start(context.Background(), "")
	messageWithSpanContext := sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo")}
	propagators.Inject(ctx, NewProducerMessageCarrier(&messageWithSpanContext))

	// Expected
	expectedList := []struct {
		attributeList []attribute.KeyValue
		parentSpanID  oteltrace.SpanID
		kind          oteltrace.SpanKind
	}{
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String(topic),
				semconv.MessagingMessageIDKey.String("1"),
				kafkaPartitionKey.Int64(0),
			},
			parentSpanID: oteltrace.SpanContextFromContext(ctx).SpanID(),
			kind:         oteltrace.SpanKindProducer,
		},
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String(topic),
				semconv.MessagingMessageIDKey.String("2"),
				kafkaPartitionKey.Int64(0),
			},
			kind: oteltrace.SpanKindProducer,
		},
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String(topic),
				// TODO: The mock sync producer of sarama does not handle the offset while sending messages
				// https://github.com/Shopify/sarama/pull/1747
				//semconv.MessagingMessageIDKey.String("3"),
				kafkaPartitionKey.Int64(0),
			},
			kind: oteltrace.SpanKindProducer,
		},
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String(topic),
				//semconv.MessagingMessageIDKey.String("4"),
				kafkaPartitionKey.Int64(0),
			},
			kind: oteltrace.SpanKindProducer,
		},
	}
	for i := 0; i < len(expectedList); i++ {
		mockSyncProducer.ExpectSendMessageAndSucceed()
	}

	// Send message
	msgList := []*sarama.ProducerMessage{
		&messageWithSpanContext,
		{Topic: topic, Key: sarama.StringEncoder("foo2")},
		{Topic: topic, Key: sarama.StringEncoder("foo3")},
		{Topic: topic, Key: sarama.StringEncoder("foo4")},
	}
	_, _, err = syncProducer.SendMessage(msgList[0])
	require.NoError(t, err)
	_, _, err = syncProducer.SendMessage(msgList[1])
	require.NoError(t, err)
	// Send messages
	require.NoError(t, syncProducer.SendMessages(msgList[2:]))

	spanList := sr.Completed()
	for i, expected := range expectedList {
		span := spanList[i]
		msg := msgList[i]

		// Check span
		assert.True(t, span.SpanContext().IsValid())
		assert.Equal(t, expected.parentSpanID, span.ParentSpanID())
		assert.Equal(t, "kafka.produce", span.Name())
		assert.Equal(t, expected.kind, span.SpanKind())
		for _, k := range expected.attributeList {
			assert.Equal(t, k.Value, span.Attributes()[k.Key], k.Key)
		}

		// Check tracing propagation
		remoteSpanFromMessage := oteltrace.SpanContextFromContext(propagators.Extract(context.Background(), NewProducerMessageCarrier(msg)))
		assert.True(t, remoteSpanFromMessage.IsValid())
	}
}

func TestWrapAsyncProducer(t *testing.T) {
	propagators := propagation.TraceContext{}
	// Create message with span context
	createMessages := func(mt oteltrace.Tracer) []*sarama.ProducerMessage {
		ctx, _ := mt.Start(context.Background(), "")
		messageWithSpanContext := sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo")}
		propagators.Inject(ctx, NewProducerMessageCarrier(&messageWithSpanContext))

		return []*sarama.ProducerMessage{
			&messageWithSpanContext,
			{Topic: topic, Key: sarama.StringEncoder("foo2")},
		}
	}

	t.Run("without successes config", func(t *testing.T) {
		// Mock provider
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(
			oteltest.WithSpanRecorder(sr),
		)

		cfg := newSaramaConfig()
		mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
		ap := WrapAsyncProducer(cfg, mockAsyncProducer, WithTracerProvider(provider), WithPropagators(propagators))

		msgList := createMessages(provider.Tracer(defaultTracerName))
		// Send message
		for _, msg := range msgList {
			mockAsyncProducer.ExpectInputAndSucceed()
			ap.Input() <- msg
		}

		err := ap.Close()
		require.NoError(t, err)

		spanList := sr.Completed()

		// Expected
		expectedList := []struct {
			attributeList []attribute.KeyValue
			parentSpanID  oteltrace.SpanID
			kind          oteltrace.SpanKind
		}{
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
				},
				parentSpanID: oteltrace.SpanID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				kind:         oteltrace.SpanKindProducer,
			},
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
				},
				kind: oteltrace.SpanKindProducer,
			},
		}
		for i, expected := range expectedList {
			span := spanList[i]
			msg := msgList[i]

			// Check span
			assert.True(t, span.SpanContext().IsValid())
			assert.Equal(t, expected.parentSpanID, span.ParentSpanID())
			assert.Equal(t, "kafka.produce", span.Name())
			assert.Equal(t, expected.kind, span.SpanKind())
			for _, k := range expected.attributeList {
				assert.Equal(t, k.Value, span.Attributes()[k.Key], k.Key)
			}

			// Check tracing propagation
			remoteSpanFromMessage := oteltrace.SpanContextFromContext(propagators.Extract(context.Background(), NewProducerMessageCarrier(msg)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	})

	t.Run("with successes config", func(t *testing.T) {
		// Mock provider
		sr := new(oteltest.SpanRecorder)
		provider := oteltest.NewTracerProvider(
			oteltest.WithSpanRecorder(sr),
		)

		// Set producer with successes config
		cfg := newSaramaConfig()
		cfg.Producer.Return.Successes = true

		mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
		ap := WrapAsyncProducer(cfg, mockAsyncProducer, WithTracerProvider(provider), WithPropagators(propagators))

		msgList := createMessages(provider.Tracer(defaultTracerName))
		// Send message
		for i, msg := range msgList {
			mockAsyncProducer.ExpectInputAndSucceed()
			// Add metadata to msg
			msg.Metadata = i
			ap.Input() <- msg
			newMsg := <-ap.Successes()
			assert.Equal(t, newMsg, msg)
		}

		err := ap.Close()
		require.NoError(t, err)

		spanList := sr.Completed()

		// Expected
		expectedList := []struct {
			attributeList []attribute.KeyValue
			parentSpanID  oteltrace.SpanID
			kind          oteltrace.SpanKind
		}{
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
					semconv.MessagingMessageIDKey.String("1"),
					kafkaPartitionKey.Int64(0),
				},
				parentSpanID: oteltrace.SpanID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				kind:         oteltrace.SpanKindProducer,
			},
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
					semconv.MessagingMessageIDKey.String("2"),
					kafkaPartitionKey.Int64(0),
				},
				kind: oteltrace.SpanKindProducer,
			},
		}
		for i, expected := range expectedList {
			span := spanList[i]
			msg := msgList[i]

			// Check span
			assert.True(t, span.SpanContext().IsValid())
			assert.Equal(t, expected.parentSpanID, span.ParentSpanID())
			assert.Equal(t, "kafka.produce", span.Name())
			assert.Equal(t, expected.kind, span.SpanKind())
			for _, k := range expected.attributeList {
				assert.Equal(t, k.Value, span.Attributes()[k.Key], k.Key)
			}

			// Check metadata
			assert.Equal(t, i, msg.Metadata)

			// Check tracing propagation
			remoteSpanFromMessage := oteltrace.SpanContextFromContext(propagators.Extract(context.Background(), NewProducerMessageCarrier(msg)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	})
}

func TestWrapAsyncProducerError(t *testing.T) {
	propagators := propagation.TraceContext{}
	// Mock provider
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	// Set producer with successes config
	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true

	mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
	ap := WrapAsyncProducer(cfg, mockAsyncProducer, WithTracerProvider(provider), WithPropagators(propagators))

	mockAsyncProducer.ExpectInputAndFail(errors.New("test"))
	ap.Input() <- &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo2")}

	err := <-ap.Errors()
	require.Error(t, err)

	ap.AsyncClose()

	spanList := sr.Completed()
	assert.Len(t, spanList, 1)

	span := spanList[0]

	assert.Equal(t, codes.Error, span.StatusCode())
	assert.Equal(t, "test", span.StatusMessage())
}

func newSaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0
	return cfg
}

func BenchmarkWrapSyncProducer(b *testing.B) {
	// Mock provider
	provider := oteltest.NewTracerProvider()

	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(b, cfg)

	// Wrap sync producer
	syncProducer := WrapSyncProducer(cfg, mockSyncProducer, WithTracerProvider(provider))
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockSyncProducer.ExpectSendMessageAndSucceed()
		_, _, err := syncProducer.SendMessage(&message)
		assert.NoError(b, err)
	}
}

func BenchmarkMockSyncProducer(b *testing.B) {
	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(b, cfg)

	// Wrap sync producer
	syncProducer := mockSyncProducer
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockSyncProducer.ExpectSendMessageAndSucceed()
		_, _, err := syncProducer.SendMessage(&message)
		assert.NoError(b, err)
	}
}

func BenchmarkWrapAsyncProducer(b *testing.B) {
	// Mock provider
	provider := oteltest.NewTracerProvider()

	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true
	mockAsyncProducer := mocks.NewAsyncProducer(b, cfg)

	// Wrap sync producer
	asyncProducer := WrapAsyncProducer(cfg, mockAsyncProducer, WithTracerProvider(provider))
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockAsyncProducer.ExpectInputAndSucceed()
		asyncProducer.Input() <- &message
		<-asyncProducer.Successes()
	}
}

func BenchmarkMockAsyncProducer(b *testing.B) {
	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true
	mockAsyncProducer := mocks.NewAsyncProducer(b, cfg)

	// Wrap sync producer
	asyncProducer := mockAsyncProducer
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockAsyncProducer.ExpectInputAndSucceed()
		mockAsyncProducer.Input() <- &message
		<-asyncProducer.Successes()
	}
}

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
	"errors"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/api/trace"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
)

func TestWrapSyncProducer(t *testing.T) {
	var err error

	// Mock tracer
	mt := mocktracer.NewTracer("kafka")

	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(t, cfg)

	// Wrap sync producer
	syncProducer := WrapSyncProducer(serviceName, cfg, mockSyncProducer, WithTracer(mt))

	// Create message with span context
	ctx, _ := mt.Start(context.Background(), "")
	messageWithSpanContext := sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo")}
	propagation.InjectHTTP(ctx, propagators, NewProducerMessageCarrier(&messageWithSpanContext))

	// Expected
	expectedList := []struct {
		kvList       []kv.KeyValue
		parentSpanID trace.SpanID
		kind         trace.SpanKind
	}{
		{
			kvList: []kv.KeyValue{
				standard.ServiceNameKey.String(serviceName),
				standard.MessagingSystemKey.String("kafka"),
				standard.MessagingDestinationKindKeyTopic,
				standard.MessagingDestinationKey.String(topic),
				standard.MessagingMessageIDKey.Int64(1),
				kafkaPartitionKey.Int32(0),
			},
			parentSpanID: trace.SpanFromContext(ctx).SpanContext().SpanID,
			kind:         trace.SpanKindProducer,
		},
		{
			kvList: []kv.KeyValue{
				standard.ServiceNameKey.String(serviceName),
				standard.MessagingSystemKey.String("kafka"),
				standard.MessagingDestinationKindKeyTopic,
				standard.MessagingDestinationKey.String(topic),
				standard.MessagingMessageIDKey.Int64(2),
				kafkaPartitionKey.Int32(0),
			},
			kind: trace.SpanKindProducer,
		},
		{
			kvList: []kv.KeyValue{
				standard.ServiceNameKey.String(serviceName),
				standard.MessagingSystemKey.String("kafka"),
				standard.MessagingDestinationKindKeyTopic,
				standard.MessagingDestinationKey.String(topic),
				// TODO: The mock sync producer of sarama does not handle the offset while sending messages
				// https://github.com/Shopify/sarama/pull/1747
				//standard.MessagingMessageIDKey.Int64(3),
				kafkaPartitionKey.Int32(0),
			},
			kind: trace.SpanKindProducer,
		},
		{
			kvList: []kv.KeyValue{
				standard.ServiceNameKey.String(serviceName),
				standard.MessagingSystemKey.String("kafka"),
				standard.MessagingDestinationKindKeyTopic,
				standard.MessagingDestinationKey.String(topic),
				//standard.MessagingMessageIDKey.Int64(4),
				kafkaPartitionKey.Int32(0),
			},
			kind: trace.SpanKindProducer,
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
	assert.NoError(t, err)
	_, _, err = syncProducer.SendMessage(msgList[1])
	assert.NoError(t, err)
	// Send messages
	assert.NoError(t, syncProducer.SendMessages(msgList[2:]))

	spanList := mt.EndedSpans()
	for i, expected := range expectedList {
		span := spanList[i]
		msg := msgList[i]

		// Check span
		assert.True(t, span.SpanContext().IsValid())
		assert.Equal(t, expected.parentSpanID, span.ParentSpanID)
		assert.Equal(t, "kafka.produce", span.Name)
		assert.Equal(t, expected.kind, span.Kind)
		for _, k := range expected.kvList {
			assert.Equal(t, k.Value, span.Attributes[k.Key], k.Key)
		}

		// Check tracing propagation
		remoteSpanFromMessage := trace.RemoteSpanContextFromContext(propagation.ExtractHTTP(context.Background(), propagators, NewProducerMessageCarrier(msg)))
		assert.True(t, remoteSpanFromMessage.IsValid())
	}
}

func TestWrapAsyncProducer(t *testing.T) {
	// Create message with span context
	createMessages := func(mt *mocktracer.Tracer) []*sarama.ProducerMessage {
		ctx, _ := mt.Start(context.Background(), "")
		messageWithSpanContext := sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo")}
		propagation.InjectHTTP(ctx, propagators, NewProducerMessageCarrier(&messageWithSpanContext))
		mt.EndedSpans()

		return []*sarama.ProducerMessage{
			&messageWithSpanContext,
			{Topic: topic, Key: sarama.StringEncoder("foo2")},
		}
	}

	t.Run("without successes config", func(t *testing.T) {
		mt := mocktracer.NewTracer("kafka")
		cfg := newSaramaConfig()
		mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
		ap := WrapAsyncProducer(serviceName, cfg, mockAsyncProducer, WithTracer(mt))

		msgList := createMessages(mt)
		// Send message
		for _, msg := range msgList {
			mockAsyncProducer.ExpectInputAndSucceed()
			ap.Input() <- msg
		}

		err := ap.Close()
		assert.NoError(t, err)

		spanList := mt.EndedSpans()

		// Expected
		expectedList := []struct {
			kvList       []kv.KeyValue
			parentSpanID trace.SpanID
			kind         trace.SpanKind
		}{
			{
				kvList: []kv.KeyValue{
					standard.ServiceNameKey.String(serviceName),
					standard.MessagingSystemKey.String("kafka"),
					standard.MessagingDestinationKindKeyTopic,
					standard.MessagingDestinationKey.String(topic),
					standard.MessagingMessageIDKey.Int64(0),
					kafkaPartitionKey.Int32(0),
				},
				parentSpanID: trace.SpanID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
				kind:         trace.SpanKindProducer,
			},
			{
				kvList: []kv.KeyValue{
					standard.ServiceNameKey.String(serviceName),
					standard.MessagingSystemKey.String("kafka"),
					standard.MessagingDestinationKindKeyTopic,
					standard.MessagingDestinationKey.String(topic),
					standard.MessagingMessageIDKey.Int64(0),
					kafkaPartitionKey.Int32(0),
				},
				kind: trace.SpanKindProducer,
			},
		}
		for i, expected := range expectedList {
			span := spanList[i]
			msg := msgList[i]

			// Check span
			assert.True(t, span.SpanContext().IsValid())
			assert.Equal(t, expected.parentSpanID, span.ParentSpanID)
			assert.Equal(t, "kafka.produce", span.Name)
			assert.Equal(t, expected.kind, span.Kind)
			for _, k := range expected.kvList {
				assert.Equal(t, k.Value, span.Attributes[k.Key], k.Key)
			}

			// Check tracing propagation
			remoteSpanFromMessage := trace.RemoteSpanContextFromContext(propagation.ExtractHTTP(context.Background(), propagators, NewProducerMessageCarrier(msg)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	})

	t.Run("with successes config", func(t *testing.T) {
		mt := mocktracer.NewTracer("kafka")

		// Set producer with successes config
		cfg := newSaramaConfig()
		cfg.Producer.Return.Successes = true

		mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
		ap := WrapAsyncProducer(serviceName, cfg, mockAsyncProducer, WithTracer(mt))

		msgList := createMessages(mt)
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
		assert.NoError(t, err)

		spanList := mt.EndedSpans()

		// Expected
		expectedList := []struct {
			kvList       []kv.KeyValue
			parentSpanID trace.SpanID
			kind         trace.SpanKind
		}{
			{
				kvList: []kv.KeyValue{
					standard.ServiceNameKey.String(serviceName),
					standard.MessagingSystemKey.String("kafka"),
					standard.MessagingDestinationKindKeyTopic,
					standard.MessagingDestinationKey.String(topic),
					standard.MessagingMessageIDKey.Int64(1),
					kafkaPartitionKey.Int32(0),
				},
				parentSpanID: trace.SpanID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
				kind:         trace.SpanKindProducer,
			},
			{
				kvList: []kv.KeyValue{
					standard.ServiceNameKey.String(serviceName),
					standard.MessagingSystemKey.String("kafka"),
					standard.MessagingDestinationKindKeyTopic,
					standard.MessagingDestinationKey.String(topic),
					standard.MessagingMessageIDKey.Int64(2),
					kafkaPartitionKey.Int32(0),
				},
				kind: trace.SpanKindProducer,
			},
		}
		for i, expected := range expectedList {
			span := spanList[i]
			msg := msgList[i]

			// Check span
			assert.True(t, span.SpanContext().IsValid())
			assert.Equal(t, expected.parentSpanID, span.ParentSpanID)
			assert.Equal(t, "kafka.produce", span.Name)
			assert.Equal(t, expected.kind, span.Kind)
			for _, k := range expected.kvList {
				assert.Equal(t, k.Value, span.Attributes[k.Key], k.Key)
			}

			// Check metadata
			assert.Equal(t, i, msg.Metadata)

			// Check tracing propagation
			remoteSpanFromMessage := trace.RemoteSpanContextFromContext(propagation.ExtractHTTP(context.Background(), propagators, NewProducerMessageCarrier(msg)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	})
}

func TestWrapAsyncProducer_Error(t *testing.T) {
	mt := mocktracer.NewTracer("kafka")

	// Set producer with successes config
	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true

	mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
	ap := WrapAsyncProducer(serviceName, cfg, mockAsyncProducer, WithTracer(mt))

	mockAsyncProducer.ExpectInputAndFail(errors.New("test"))
	ap.Input() <- &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo2")}

	err := <-ap.Errors()
	assert.Error(t, err)

	ap.AsyncClose()

	spanList := mt.EndedSpans()
	assert.Len(t, spanList, 1)

	span := spanList[0]

	assert.Equal(t, codes.Internal, span.Status)
	assert.Equal(t, "test", span.StatusMessage)
}

func newSaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0
	return cfg
}

func BenchmarkWrapSyncProducer(b *testing.B) {
	// Mock tracer
	mt := mocktracer.NewTracer("kafka")

	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(b, cfg)

	// Wrap sync producer
	syncProducer := WrapSyncProducer(serviceName, cfg, mockSyncProducer, WithTracer(mt))
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
	// Mock tracer
	mt := mocktracer.NewTracer("kafka")

	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true
	mockAsyncProducer := mocks.NewAsyncProducer(b, cfg)

	// Wrap sync producer
	asyncProducer := WrapAsyncProducer(serviceName, cfg, mockAsyncProducer, WithTracer(mt))
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

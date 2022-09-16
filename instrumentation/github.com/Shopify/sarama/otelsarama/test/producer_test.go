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

package test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestWrapSyncProducer(t *testing.T) {
	propagators := propagation.TraceContext{}
	var err error

	// Mock provider
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(t, cfg)

	// Wrap sync producer
	syncProducer := otelsarama.WrapSyncProducer(cfg, mockSyncProducer, otelsarama.WithTracerProvider(provider), otelsarama.WithPropagators(propagators))

	// Create message with span context
	ctx, _ := provider.Tracer("test").Start(context.Background(), "")
	messageWithSpanContext := sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo")}
	propagators.Inject(ctx, otelsarama.NewProducerMessageCarrier(&messageWithSpanContext))

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
				semconv.MessagingKafkaPartitionKey.Int64(0),
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
				semconv.MessagingKafkaPartitionKey.Int64(0),
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
				semconv.MessagingKafkaPartitionKey.Int64(12),
			},
			kind: oteltrace.SpanKindProducer,
		},
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String(topic),
				//semconv.MessagingMessageIDKey.String("4"),
				semconv.MessagingKafkaPartitionKey.Int64(25),
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

	spanList := sr.Ended()
	for i, expected := range expectedList {
		span := spanList[i]
		msg := msgList[i]

		// Check span
		assert.True(t, span.SpanContext().IsValid())
		assert.Equal(t, expected.parentSpanID, span.Parent().SpanID())
		assert.Equal(t, fmt.Sprintf("%s send", topic), span.Name())
		assert.Equal(t, expected.kind, span.SpanKind())
		for _, k := range expected.attributeList {
			assert.Contains(t, span.Attributes(), k)
		}

		// Check tracing propagation
		remoteSpanFromMessage := oteltrace.SpanContextFromContext(propagators.Extract(context.Background(), otelsarama.NewProducerMessageCarrier(msg)))
		assert.True(t, remoteSpanFromMessage.IsValid())
	}
}

func TestWrapAsyncProducer(t *testing.T) {
	propagators := propagation.TraceContext{}
	// Create message with span context
	createMessages := func(mt oteltrace.Tracer) []*sarama.ProducerMessage {
		ctx, _ := mt.Start(context.Background(), "")
		messageWithSpanContext := sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo")}
		propagators.Inject(ctx, otelsarama.NewProducerMessageCarrier(&messageWithSpanContext))

		return []*sarama.ProducerMessage{
			&messageWithSpanContext,
			{Topic: topic, Key: sarama.StringEncoder("foo2")},
		}
	}

	t.Run("without successes config", func(t *testing.T) {
		// Mock provider
		sr := tracetest.NewSpanRecorder()
		provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

		cfg := newSaramaConfig()
		mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
		ap := otelsarama.WrapAsyncProducer(cfg, mockAsyncProducer, otelsarama.WithTracerProvider(provider), otelsarama.WithPropagators(propagators))

		msgList := createMessages(provider.Tracer("test"))
		// Send message
		for _, msg := range msgList {
			mockAsyncProducer.ExpectInputAndSucceed()
			ap.Input() <- msg
		}

		err := ap.Close()
		require.NoError(t, err)

		spanList := sr.Ended()

		// Expected
		expectedList := []struct {
			attributeList []attribute.KeyValue
			kind          oteltrace.SpanKind
		}{
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
				},
				kind: oteltrace.SpanKindProducer,
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
			assert.Equal(t, fmt.Sprintf("%s send", topic), span.Name())
			assert.Equal(t, expected.kind, span.SpanKind())
			for _, k := range expected.attributeList {
				assert.Contains(t, span.Attributes(), k)
			}

			// Check tracing propagation
			remoteSpanFromMessage := oteltrace.SpanContextFromContext(propagators.Extract(context.Background(), otelsarama.NewProducerMessageCarrier(msg)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	})

	t.Run("with successes config", func(t *testing.T) {
		// Mock provider
		sr := tracetest.NewSpanRecorder()
		provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

		// Set producer with successes config
		cfg := newSaramaConfig()
		cfg.Producer.Return.Successes = true

		mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
		ap := otelsarama.WrapAsyncProducer(cfg, mockAsyncProducer, otelsarama.WithTracerProvider(provider), otelsarama.WithPropagators(propagators))

		msgList := createMessages(provider.Tracer("test"))
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

		spanList := sr.Ended()

		// Expected
		expectedList := []struct {
			attributeList []attribute.KeyValue
			kind          oteltrace.SpanKind
		}{
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
					semconv.MessagingMessageIDKey.String("1"),
					semconv.MessagingKafkaPartitionKey.Int64(9),
				},
				kind: oteltrace.SpanKindProducer,
			},
			{
				attributeList: []attribute.KeyValue{
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingDestinationKey.String(topic),
					semconv.MessagingMessageIDKey.String("2"),
					semconv.MessagingKafkaPartitionKey.Int64(31),
				},
				kind: oteltrace.SpanKindProducer,
			},
		}
		for i, expected := range expectedList {
			span := spanList[i]
			msg := msgList[i]

			// Check span
			assert.True(t, span.SpanContext().IsValid())
			assert.Equal(t, fmt.Sprintf("%s send", topic), span.Name())
			assert.Equal(t, expected.kind, span.SpanKind())
			for _, k := range expected.attributeList {
				assert.Contains(t, span.Attributes(), k)
			}

			// Check metadata
			assert.Equal(t, i, msg.Metadata)

			// Check tracing propagation
			remoteSpanFromMessage := oteltrace.SpanContextFromContext(propagators.Extract(context.Background(), otelsarama.NewProducerMessageCarrier(msg)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	})
}

func TestWrapAsyncProducerError(t *testing.T) {
	propagators := propagation.TraceContext{}
	// Mock provider
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	// Set producer with successes config
	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true

	mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
	ap := otelsarama.WrapAsyncProducer(cfg, mockAsyncProducer, otelsarama.WithTracerProvider(provider), otelsarama.WithPropagators(propagators))

	mockAsyncProducer.ExpectInputAndFail(errors.New("test"))
	metadata := "test metadata"
	ap.Input() <- &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo2"), Metadata: metadata}

	err := <-ap.Errors()
	require.Error(t, err)
	assert.Equal(t, metadata, err.Msg.Metadata, "should preseve metadata")

	ap.AsyncClose()

	spanList := sr.Ended()
	assert.Len(t, spanList, 1)

	span := spanList[0]

	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, "test", span.Status().Description)
}

func TestWrapAsyncProducer_DrainsSuccessesAndErrorsChannels(t *testing.T) {
	// Mock provider
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	// Set producer with successes config and fill it with successes and errors
	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true

	mockAsyncProducer := mocks.NewAsyncProducer(t, cfg)
	ap := otelsarama.WrapAsyncProducer(cfg, mockAsyncProducer, otelsarama.WithTracerProvider(provider))

	wantSuccesses := 5
	for i := 0; i < wantSuccesses; i++ {
		mockAsyncProducer.ExpectInputAndSucceed()
		ap.Input() <- &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo2")}
	}

	wantErrros := 3
	for i := 0; i < wantErrros; i++ {
		mockAsyncProducer.ExpectInputAndFail(errors.New("test"))
		ap.Input() <- &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder("foo2")}
	}

	ap.AsyncClose()

	// Ensure it is possible to read Successes and Errors after AsyncClose
	var wg sync.WaitGroup

	gotSuccesses := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ap.Successes() {
			gotSuccesses++
		}
	}()

	gotErrors := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ap.Errors() {
			gotErrors++
		}
	}()

	wg.Wait()
	spanList := sr.Ended()
	assert.Equal(t, wantSuccesses, gotSuccesses, "should read all successes")
	assert.Equal(t, wantErrros, gotErrors, "should read all errors")
	assert.Len(t, spanList, wantSuccesses+wantErrros, "should record all spans")
}

func newSaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0
	return cfg
}

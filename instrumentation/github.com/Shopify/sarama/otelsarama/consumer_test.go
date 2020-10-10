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
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	otelpropagators "go.opentelemetry.io/otel/propagators"
	"go.opentelemetry.io/otel/semconv"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
)

const (
	topic = "test-topic"
)

func TestWrapPartitionConsumer(t *testing.T) {
	propagators := otelpropagators.TraceContext{}
	// Mock provider
	provider, mt := NewTracerProviderAndTracer()

	// Mock partition consumer controller
	consumer := mocks.NewConsumer(t, sarama.NewConfig())
	mockPartitionConsumer := consumer.ExpectConsumePartition(topic, 0, 0)

	// Create partition consumer
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	require.NoError(t, err)

	partitionConsumer = WrapPartitionConsumer(partitionConsumer, WithTracerProvider(provider), WithPropagators(propagators))

	consumeAndCheck(t, mt, mockPartitionConsumer, partitionConsumer)
}

func TestWrapConsumer(t *testing.T) {
	propagators := otelpropagators.TraceContext{}
	// Mock provider
	provider, mt := NewTracerProviderAndTracer()

	// Mock partition consumer controller
	mockConsumer := mocks.NewConsumer(t, sarama.NewConfig())
	mockPartitionConsumer := mockConsumer.ExpectConsumePartition(topic, 0, 0)

	// Wrap consumer
	consumer := WrapConsumer(mockConsumer, WithTracerProvider(provider), WithPropagators(propagators))

	// Create partition consumer
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	require.NoError(t, err)

	consumeAndCheck(t, mt, mockPartitionConsumer, partitionConsumer)
}

func consumeAndCheck(t *testing.T, mt *mocktracer.Tracer, mockPartitionConsumer *mocks.PartitionConsumer, partitionConsumer sarama.PartitionConsumer) {
	// Create message with span context
	ctx, _ := mt.Start(context.Background(), "")
	message := sarama.ConsumerMessage{Key: []byte("foo")}
	propagators := otelpropagators.TraceContext{}
	propagators.Inject(ctx, NewConsumerMessageCarrier(&message))

	// Produce message
	mockPartitionConsumer.YieldMessage(&message)
	mockPartitionConsumer.YieldMessage(&sarama.ConsumerMessage{Key: []byte("foo2")})

	// Consume messages
	msgList := make([]*sarama.ConsumerMessage, 2)
	msgList[0] = <-partitionConsumer.Messages()
	msgList[1] = <-partitionConsumer.Messages()
	require.NoError(t, partitionConsumer.Close())
	// Wait for the channel to be closed
	<-partitionConsumer.Messages()

	// Check spans length
	spans := mt.EndedSpans()
	assert.Len(t, spans, 2)

	expectedList := []struct {
		labelList    []label.KeyValue
		parentSpanID trace.SpanID
		kind         trace.SpanKind
		msgKey       []byte
	}{
		{
			labelList: []label.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindKeyTopic,
				semconv.MessagingDestinationKey.String("test-topic"),
				semconv.MessagingOperationReceive,
				semconv.MessagingMessageIDKey.String("1"),
				kafkaPartitionKey.Int32(0),
			},
			parentSpanID: trace.SpanFromContext(ctx).SpanContext().SpanID,
			kind:         trace.SpanKindConsumer,
			msgKey:       []byte("foo"),
		},
		{
			labelList: []label.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindKeyTopic,
				semconv.MessagingDestinationKey.String("test-topic"),
				semconv.MessagingOperationReceive,
				semconv.MessagingMessageIDKey.String("2"),
				kafkaPartitionKey.Int32(0),
			},
			kind:   trace.SpanKindConsumer,
			msgKey: []byte("foo2"),
		},
	}

	for i, expected := range expectedList {
		t.Run(fmt.Sprint("index", i), func(t *testing.T) {
			span := spans[i]

			assert.Equal(t, expected.parentSpanID, span.ParentSpanID)

			remoteSpanFromMessage := trace.RemoteSpanContextFromContext(propagators.Extract(context.Background(), NewConsumerMessageCarrier(msgList[i])))
			assert.Equal(t, span.SpanContext(), remoteSpanFromMessage,
				"span context should be injected into the consumer message headers")

			assert.Equal(t, "kafka.consume", span.Name)
			assert.Equal(t, expected.kind, span.Kind)
			assert.Equal(t, expected.msgKey, msgList[i].Key)
			for _, k := range expected.labelList {
				assert.Equal(t, k.Value, span.Attributes[k.Key], k.Key)
			}
		})
	}
}

func TestConsumerConsumePartitionWithError(t *testing.T) {
	// Mock partition consumer controller
	mockConsumer := mocks.NewConsumer(t, sarama.NewConfig())
	mockConsumer.ExpectConsumePartition(topic, 0, 0)

	consumer := WrapConsumer(mockConsumer)
	_, err := consumer.ConsumePartition(topic, 0, 0)
	assert.NoError(t, err)
	// Consume twice
	_, err = consumer.ConsumePartition(topic, 0, 0)
	assert.Error(t, err)
}

func BenchmarkWrapPartitionConsumer(b *testing.B) {
	// Mock provider
	provider, _ := NewTracerProviderAndTracer()

	mockPartitionConsumer, partitionConsumer := createMockPartitionConsumer(b)

	partitionConsumer = WrapPartitionConsumer(partitionConsumer, WithTracerProvider(provider))
	message := sarama.ConsumerMessage{Key: []byte("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPartitionConsumer.YieldMessage(&message)
		<-partitionConsumer.Messages()
	}
}

func BenchmarkMockPartitionConsumer(b *testing.B) {
	mockPartitionConsumer, partitionConsumer := createMockPartitionConsumer(b)

	message := sarama.ConsumerMessage{Key: []byte("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPartitionConsumer.YieldMessage(&message)
		<-partitionConsumer.Messages()
	}
}

func createMockPartitionConsumer(b *testing.B) (*mocks.PartitionConsumer, sarama.PartitionConsumer) {
	// Mock partition consumer controller
	consumer := mocks.NewConsumer(b, sarama.NewConfig())
	mockPartitionConsumer := consumer.ExpectConsumePartition(topic, 0, 0)

	// Create partition consumer
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	require.NoError(b, err)
	return mockPartitionConsumer, partitionConsumer
}

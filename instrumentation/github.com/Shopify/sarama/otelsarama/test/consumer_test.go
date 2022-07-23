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
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	topic = "test-topic"
)

func TestWrapPartitionConsumer(t *testing.T) {
	propagators := propagation.TraceContext{}
	// Mock provider
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// Mock partition consumer controller
	consumer := mocks.NewConsumer(t, sarama.NewConfig())
	mockPartitionConsumer := consumer.ExpectConsumePartition(topic, 0, 0)

	// Create partition consumer
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	require.NoError(t, err)

	partitionConsumer = otelsarama.WrapPartitionConsumer(partitionConsumer, otelsarama.WithTracerProvider(provider), otelsarama.WithPropagators(propagators))

	consumeAndCheck(t, provider.Tracer("test"), sr.Ended, mockPartitionConsumer, partitionConsumer)
}

func TestWrapConsumer(t *testing.T) {
	propagators := propagation.TraceContext{}
	// Mock provider
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// Mock partition consumer controller
	mockConsumer := mocks.NewConsumer(t, sarama.NewConfig())
	mockPartitionConsumer := mockConsumer.ExpectConsumePartition(topic, 0, 0)

	// Wrap consumer
	consumer := otelsarama.WrapConsumer(mockConsumer, otelsarama.WithTracerProvider(provider), otelsarama.WithPropagators(propagators))

	// Create partition consumer
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	require.NoError(t, err)

	consumeAndCheck(t, provider.Tracer("test"), sr.Ended, mockPartitionConsumer, partitionConsumer)
}

func consumeAndCheck(t *testing.T, mt trace.Tracer, complFn func() []sdktrace.ReadOnlySpan, mockPartitionConsumer *mocks.PartitionConsumer, partitionConsumer sarama.PartitionConsumer) {
	// Create message with span context
	ctx, _ := mt.Start(context.Background(), "")
	message := sarama.ConsumerMessage{Key: []byte("foo")}
	propagators := propagation.TraceContext{}
	propagators.Inject(ctx, otelsarama.NewConsumerMessageCarrier(&message))

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
	spans := complFn()
	assert.Len(t, spans, 2)

	expectedList := []struct {
		attributeList []attribute.KeyValue
		parentSpanID  trace.SpanID
		kind          trace.SpanKind
		msgKey        []byte
	}{
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String("test-topic"),
				semconv.MessagingOperationReceive,
				semconv.MessagingMessageIDKey.String("0"),
				semconv.MessagingKafkaPartitionKey.Int64(0),
			},
			parentSpanID: trace.SpanContextFromContext(ctx).SpanID(),
			kind:         trace.SpanKindConsumer,
			msgKey:       []byte("foo"),
		},
		{
			attributeList: []attribute.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindTopic,
				semconv.MessagingDestinationKey.String("test-topic"),
				semconv.MessagingOperationReceive,
				semconv.MessagingMessageIDKey.String("1"),
				semconv.MessagingKafkaPartitionKey.Int64(0),
			},
			kind:   trace.SpanKindConsumer,
			msgKey: []byte("foo2"),
		},
	}

	for i, expected := range expectedList {
		t.Run(fmt.Sprint("index", i), func(t *testing.T) {
			span := spans[i]

			assert.Equal(t, expected.parentSpanID, span.Parent().SpanID())

			sc := trace.SpanContextFromContext(propagators.Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(msgList[i])))
			// propagators.Extract always returns a remote SpanContext.
			assert.Equal(t, sc, span.SpanContext().WithRemote(true))

			assert.Equal(t, fmt.Sprintf("%s receive", topic), span.Name())
			assert.Equal(t, expected.kind, span.SpanKind())
			assert.Equal(t, expected.msgKey, msgList[i].Key)
			for _, k := range expected.attributeList {
				assert.Contains(t, span.Attributes(), k)
			}
		})
	}
}

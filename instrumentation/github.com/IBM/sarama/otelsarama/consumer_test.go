// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newTestConsumer(t *testing.T) (*mocks.Consumer, *tracetest.SpanRecorder, sarama.Consumer) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	mc := mocks.NewConsumer(t, nil)
	wrapped := WrapConsumer(mc, WithTracerProvider(tp))
	return mc, sr, wrapped
}

func TestWrapPartitionConsumerCreatesSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	mc := mocks.NewConsumer(t, nil)
	pmc := mc.ExpectConsumePartition("test-topic", 0, sarama.OffsetOldest)
	pmc.YieldMessage(&sarama.ConsumerMessage{Topic: "test-topic", Partition: 0})

	rawPC, err := mc.ConsumePartition("test-topic", 0, sarama.OffsetOldest)
	require.NoError(t, err)

	wrapped := WrapPartitionConsumer(rawPC, WithTracerProvider(tp))

	msg := <-wrapped.Messages()
	assert.NotNil(t, msg)
	assert.Equal(t, "test-topic", msg.Topic)

	rawPC.AsyncClose()
	<-wrapped.Messages() // drain until closed

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "process test-topic", span.Name())

	attrs := spanAttrs(span)
	assert.Equal(t, "kafka", attrs["messaging.system"])
	assert.Equal(t, "test-topic", attrs["messaging.destination.name"])
	assert.Equal(t, "process", attrs["messaging.operation.name"])
	assert.Equal(t, "process", attrs["messaging.operation.type"])
	assert.Equal(t, "0", attrs["messaging.destination.partition.id"])
	assert.Equal(t, int64(0), attrs["messaging.kafka.offset"])
}

func TestWrapConsumerConsumePartitionCreatesSpan(t *testing.T) {
	_, sr, wrapped := newTestConsumer(t)

	mc := wrapped.(*consumer).Consumer.(*mocks.Consumer)
	pmc := mc.ExpectConsumePartition("topic", 0, sarama.OffsetOldest)
	pmc.YieldMessage(&sarama.ConsumerMessage{Topic: "topic", Partition: 0})

	pc, err := wrapped.ConsumePartition("topic", 0, sarama.OffsetOldest)
	require.NoError(t, err)

	msg := <-pc.Messages()
	assert.NotNil(t, msg)

	pc.(*partitionConsumer).PartitionConsumer.AsyncClose()
	<-pc.Messages()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "process topic", spans[0].Name())
}

func TestConsumerSpanWithClusterID(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	mc := mocks.NewConsumer(t, nil)
	pmc := mc.ExpectConsumePartition("test-topic", 0, sarama.OffsetOldest)
	pmc.YieldMessage(&sarama.ConsumerMessage{Topic: "test-topic", Partition: 0})

	rawPC, err := mc.ConsumePartition("test-topic", 0, sarama.OffsetOldest)
	require.NoError(t, err)

	cfg := newConfig(WithTracerProvider(tp))
	clusterID := "my-cluster"
	cfg.clusterID.Store(&clusterID)

	dispatcher := newConsumerMessagesDispatcherWrapper(rawPC, cfg)
	go dispatcher.Run()

	msg := <-dispatcher.Messages()
	assert.NotNil(t, msg)

	rawPC.AsyncClose()
	<-dispatcher.Messages()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	attrs := spanAttrs(spans[0])
	assert.Equal(t, "my-cluster", attrs["messaging.kafka.cluster.id"])
}

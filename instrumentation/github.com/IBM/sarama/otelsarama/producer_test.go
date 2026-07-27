// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"errors"
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func newTestSyncProducer(t *testing.T) (*mocks.SyncProducer, *tracetest.SpanRecorder, sarama.SyncProducer) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V0_11_0_0
	mp := mocks.NewSyncProducer(t, saramaConfig)

	wrapped := WrapSyncProducer(saramaConfig, mp, WithTracerProvider(tp), WithPropagators(prop))
	return mp, sr, wrapped
}

func TestSyncProducerSendMessage(t *testing.T) {
	mp, sr, wrapped := newTestSyncProducer(t)
	mp.ExpectSendMessageAndSucceed()

	msg := &sarama.ProducerMessage{Topic: "test-topic"}
	partition, offset, err := wrapped.SendMessage(msg)
	require.NoError(t, err)
	assert.Equal(t, int32(0), partition)
	assert.Equal(t, int64(1), offset)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "send test-topic", span.Name())
	assert.Equal(t, trace.SpanKindProducer, span.SpanKind())
	assert.Equal(t, codes.Unset, span.Status().Code)

	attrs := spanAttrs(span)
	assert.Equal(t, "kafka", attrs["messaging.system"])
	assert.Equal(t, "test-topic", attrs["messaging.destination.name"])
	assert.Equal(t, "send", attrs["messaging.operation.name"])
	assert.Equal(t, "send", attrs["messaging.operation.type"])
}

func TestSyncProducerSendMessageError(t *testing.T) {
	mp, sr, wrapped := newTestSyncProducer(t)
	mp.ExpectSendMessageAndFail(errors.New("broker unavailable"))

	msg := &sarama.ProducerMessage{Topic: "test-topic"}
	_, _, err := wrapped.SendMessage(msg)
	assert.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code)
}

func TestSyncProducerSendMessages(t *testing.T) {
	mp, sr, wrapped := newTestSyncProducer(t)
	mp.ExpectSendMessageAndSucceed()
	mp.ExpectSendMessageAndSucceed()

	msgs := []*sarama.ProducerMessage{
		{Topic: "topic-a"},
		{Topic: "topic-b"},
	}
	err := wrapped.SendMessages(msgs)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 2)
}

func TestSyncProducerInjectsTraceContext(t *testing.T) {
	mp, _, wrapped := newTestSyncProducer(t)
	mp.ExpectSendMessageAndSucceed()

	msg := &sarama.ProducerMessage{Topic: "test-topic"}
	_, _, err := wrapped.SendMessage(msg)
	require.NoError(t, err)

	// Headers should contain traceparent or similar propagation headers.
	assert.NotEmpty(t, msg.Headers, "expected trace context headers to be injected")
}

func TestSyncProducerWithClusterID(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V0_11_0_0
	mp := mocks.NewSyncProducer(t, saramaConfig)
	mp.ExpectSendMessageAndSucceed()

	// Provide cluster ID via config (bypass WithClient by directly setting atomic).
	cfg := newConfig(WithTracerProvider(tp))
	clusterID := "test-cluster-id"
	cfg.clusterID.Store(&clusterID)

	wrapped := &syncProducer{
		SyncProducer: mp,
		cfg:          cfg,
		saramaConfig: saramaConfig,
	}

	msg := &sarama.ProducerMessage{Topic: "test-topic"}
	_, _, err := wrapped.SendMessage(msg)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	attrs := spanAttrs(spans[0])
	assert.Equal(t, "test-cluster-id", attrs["messaging.kafka.cluster.id"])
}

// spanAttrs returns a map of attribute key→value for a recorded span.
func spanAttrs(span sdktrace.ReadOnlySpan) map[string]any {
	m := make(map[string]any, len(span.Attributes()))
	for _, kv := range span.Attributes() {
		m[string(kv.Key)] = kv.Value.AsInterface()
	}
	return m
}

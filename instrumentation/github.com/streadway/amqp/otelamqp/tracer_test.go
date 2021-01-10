package otelamqp

import (
	"context"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/oteltest"
	"testing"
)

func TestStartConsumerSpan(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	hdrs := amqp.Table{}
	consumerSpan, _ := StartConsumerSpan(hdrs, context.Background())
	_, ok := consumerSpan.(*oteltest.Span)
	assert.True(t, ok)
	spanTracer := consumerSpan.Tracer()
	mockTracer, ok := spanTracer.(*oteltest.Tracer)
	require.True(t, ok)
	assert.Equal(t, "amqp", mockTracer.Name)
}

func TestStartProducerSpan(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	hdrs := amqp.Table{}
	producerSpan := StartProducerSpan(hdrs, context.Background())
	_, ok := producerSpan.(*oteltest.Span)
	assert.True(t, ok)
	spanTracer := producerSpan.Tracer()
	mockTracer, ok := spanTracer.(*oteltest.Tracer)
	require.True(t, ok)
	assert.Equal(t, "amqp", mockTracer.Name)
}


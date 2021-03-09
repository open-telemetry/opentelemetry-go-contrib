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

package otelamqp

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel/codes"
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/oteltest"
)

func TestStartConsumerSpan(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	hdrs := amqp.Table{}
	consumerSpan, _ := StartConsumerSpan(context.Background(), hdrs)
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
	producerSpan := StartProducerSpan(context.Background(), hdrs)
	_, ok := producerSpan.(*oteltest.Span)
	assert.True(t, ok)
	spanTracer := producerSpan.Tracer()
	mockTracer, ok := spanTracer.(*oteltest.Tracer)
	require.True(t, ok)
	assert.Equal(t, "amqp", mockTracer.Name)
}

func TestEndConsumerSpan(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())
	hdrs := amqp.Table{}
	consumerSpan, _ := StartConsumerSpan(context.Background(), hdrs)

	EndConsumerSpan(consumerSpan, nil)
	span, ok := consumerSpan.(*oteltest.Span)
	assert.True(t, ok)
	spanTracer := consumerSpan.Tracer()
	mockTracer, ok := spanTracer.(*oteltest.Tracer)
	require.True(t, ok)
	require.True(t, span.Ended())
	assert.Equal(t, "amqp", mockTracer.Name)
}

func TestEndConsumerSpanWhenError(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())
	hdrs := amqp.Table{}
	consumerSpan, _ := StartConsumerSpan(context.Background(), hdrs)

	EndConsumerSpan(consumerSpan, errors.New("error"))
	span, ok := consumerSpan.(*oteltest.Span)
	assert.True(t, ok)
	assert.Equal(t, codes.Error, span.StatusCode())
}

func TestEndProducerSpan(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())
	hdrs := amqp.Table{}
	producerSpan := StartProducerSpan(context.Background(), hdrs)

	EndConsumerSpan(producerSpan, nil)
	span, ok := producerSpan.(*oteltest.Span)
	assert.True(t, ok)
	spanTracer := producerSpan.Tracer()
	mockTracer, ok := spanTracer.(*oteltest.Tracer)
	require.True(t, ok)
	require.True(t, span.Ended())
	assert.Equal(t, "amqp", mockTracer.Name)
}

func TestEndProducerSpanWhenError(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())
	hdrs := amqp.Table{}
	consumerSpan := StartProducerSpan(context.Background(), hdrs)

	EndProducerSpan(consumerSpan, errors.New("error"))
	span, ok := consumerSpan.(*oteltest.Span)
	assert.True(t, ok)
	assert.Equal(t, codes.Error, span.StatusCode())
}

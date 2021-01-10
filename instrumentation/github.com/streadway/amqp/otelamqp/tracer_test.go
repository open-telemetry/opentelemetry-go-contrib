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


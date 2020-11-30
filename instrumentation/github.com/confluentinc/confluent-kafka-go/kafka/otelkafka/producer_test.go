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

package otelkafka

import (
	"context"
	"testing"
	"time"

	mocktracer "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagators"
	"go.opentelemetry.io/otel/semconv"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
)

// NewTracerProviderAndTracer returns mock traceProvider and tracer
func NewTracerProviderAndTracer() (*mocktracer.TracerProvider, *mocktracer.Tracer) {
	var provider mocktracer.TracerProvider
	tracer := provider.Tracer(tracerName)
	return &provider, tracer.(*mocktracer.Tracer)
}

// TestProducerAPIs dry-tests all Producer APIs, no broker is needed.
func TestProducer_Produce(t *testing.T) {
	var (
		drChan    = make(chan kafka.Event, 2)
		topic1    = "kafka-test-topic1"
		topic2    = "kafka-test-topic2"
		expMsgCnt = 0
		msgs      = []struct {
			Key   string
			Value string
		}{
			{"This is my key1", "Value1"},
			{"This is my key2", "Value2"}}
	)

	traceProvider, tracer := NewTracerProviderAndTracer()

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"socket.timeout.ms":  10,
		"message.timeout.ms": 10})
	assert.Nil(t, err)

	producer := WrapProducer(p, WithContext(context.Background()),
		WithTracerProvider(traceProvider),
		WithTracer(tracer),
		WithPropagators(otel.NewCompositeTextMapPropagator(propagators.TraceContext{}, propagators.Baggage{})),
	)

	// Produce  message
	err = producer.Produce(&kafka.Message{TopicPartition: kafka.TopicPartition{Topic: &topic1, Partition: 0},
		Value: []byte(msgs[0].Value), Key: []byte(msgs[0].Key)},
		drChan)
	assert.Nil(t, err)
	expMsgCnt++

	time.Sleep(time.Microsecond)

	// Produce  message with span context
	ctx, _ := tracer.Start(context.Background(), "")
	messageWithSpanContext := &kafka.Message{TopicPartition: kafka.TopicPartition{Topic: &topic2, Partition: 0},
		Value: []byte(msgs[1].Value), Key: []byte(msgs[1].Key)}
	producer.cfg.Propagators.Inject(ctx, NewMessageCarrier(messageWithSpanContext))

	err = producer.Produce(messageWithSpanContext,
		drChan)
	assert.Nil(t, err)
	expMsgCnt++

	// Check for produced message count
	assert.Equal(t, expMsgCnt, producer.Len())

	// Expected Span attributes fo t
	expectedList := []struct {
		labelList    []label.KeyValue
		parentSpanID trace.SpanID
		kind         trace.SpanKind
		topic        string
	}{
		{
			labelList: []label.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindKeyTopic,
				semconv.MessagingDestinationKey.String(topic1),
				label.Key(kafkaPartitionField).Int32(0),
				label.Key(kafkaMessageKeyField).String(msgs[0].Key),
			},
			kind:  trace.SpanKindProducer,
			topic: topic1,
		},
		{
			labelList: []label.KeyValue{
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationKindKeyTopic,
				semconv.MessagingDestinationKey.String(topic2),
				label.Key(kafkaPartitionField).Int32(0),
				label.Key(kafkaMessageKeyField).String(msgs[1].Key),
			},
			parentSpanID: trace.SpanFromContext(ctx).SpanContext().SpanID,
			kind:         trace.SpanKindProducer,
			topic:        topic2,
		},
	}

	// Wait some time to end spanned spans
	time.Sleep(time.Second * 1)

	spanList := tracer.EndedSpans()
	for i, expected := range expectedList {
		span := spanList[i]

		// Check span
		assert.True(t, span.SpanContext().IsValid())
		assert.Equal(t, expected.kind, span.Kind)
		assert.Equal(t, expected.topic+" send", span.Name)

		for _, k := range expected.labelList {
			assert.Equal(t, k.Value, span.Attributes[k.Key], k.Key)
		}

		if expected.topic == topic2 { // Check for remote span context propagation
			assert.Equal(t, expected.parentSpanID, span.ParentSpanID)
			remoteSpanFromMessage := trace.RemoteSpanContextFromContext(producer.cfg.Propagators.Extract(context.Background(), NewMessageCarrier(messageWithSpanContext)))
			assert.True(t, remoteSpanFromMessage.IsValid())
		}
	}
}

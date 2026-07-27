// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func TestProducerMessageCarrierGet(t *testing.T) {
	msg := &sarama.ProducerMessage{
		Headers: []sarama.RecordHeader{
			{Key: []byte("traceparent"), Value: []byte("val1")},
			{Key: []byte("other"), Value: []byte("val2")},
		},
	}
	c := NewProducerMessageCarrier(msg)
	assert.Equal(t, "val1", c.Get("traceparent"))
	assert.Empty(t, c.Get("missing"))
}

func TestProducerMessageCarrierSet(t *testing.T) {
	msg := &sarama.ProducerMessage{}
	c := NewProducerMessageCarrier(msg)

	c.Set("traceparent", "val1")
	assert.Equal(t, "val1", c.Get("traceparent"))

	// Deduplication: setting same key updates in-place.
	c.Set("traceparent", "val2")
	assert.Equal(t, "val2", c.Get("traceparent"))
	assert.Len(t, msg.Headers, 1, "should not append duplicate key")
}

func TestProducerMessageCarrierKeys(t *testing.T) {
	msg := &sarama.ProducerMessage{
		Headers: []sarama.RecordHeader{
			{Key: []byte("a")},
			{Key: []byte("b")},
		},
	}
	c := NewProducerMessageCarrier(msg)
	assert.Equal(t, []string{"a", "b"}, c.Keys())
}

func TestConsumerMessageCarrierGet(t *testing.T) {
	msg := &sarama.ConsumerMessage{
		Headers: []*sarama.RecordHeader{
			{Key: []byte("traceparent"), Value: []byte("val1")},
			nil,
		},
	}
	c := NewConsumerMessageCarrier(msg)
	assert.Equal(t, "val1", c.Get("traceparent"))
	assert.Empty(t, c.Get("missing"))
}

func TestConsumerMessageCarrierKeys(t *testing.T) {
	msg := &sarama.ConsumerMessage{
		Headers: []*sarama.RecordHeader{
			{Key: []byte("a")},
			nil,
			{Key: []byte("b")},
		},
	}
	c := NewConsumerMessageCarrier(msg)
	assert.Equal(t, []string{"a", "b"}, c.Keys())
}

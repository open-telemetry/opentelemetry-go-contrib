// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"github.com/IBM/sarama"
)

// ProducerMessageCarrier injects and extracts traces from a sarama ProducerMessage.
type ProducerMessageCarrier struct {
	msg *sarama.ProducerMessage
}

// NewProducerMessageCarrier creates a new ProducerMessageCarrier.
func NewProducerMessageCarrier(msg *sarama.ProducerMessage) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

// Get returns the value associated with the passed key.
func (c ProducerMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set stores the key-value pair. It deduplicates by replacing any existing header with the same key.
func (c ProducerMessageCarrier) Set(key, val string) {
	for i, h := range c.msg.Headers {
		if string(h.Key) == key {
			c.msg.Headers[i].Value = []byte(val)
			return
		}
	}
	c.msg.Headers = append(c.msg.Headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

// Keys lists the keys stored in this carrier.
func (c ProducerMessageCarrier) Keys() []string {
	keys := make([]string, len(c.msg.Headers))
	for i, h := range c.msg.Headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// ConsumerMessageCarrier extracts traces from a sarama ConsumerMessage.
type ConsumerMessageCarrier struct {
	msg *sarama.ConsumerMessage
}

// NewConsumerMessageCarrier creates a new ConsumerMessageCarrier.
func NewConsumerMessageCarrier(msg *sarama.ConsumerMessage) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

// Get returns the value associated with the passed key.
func (c ConsumerMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h != nil && string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set is a no-op for consumer messages; trace context propagation is
// injected on the producer side.
func (c ConsumerMessageCarrier) Set(string, string) {}

// Keys lists the keys stored in this carrier.
func (c ConsumerMessageCarrier) Keys() []string {
	keys := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		if h != nil {
			keys = append(keys, string(h.Key))
		}
	}
	return keys
}

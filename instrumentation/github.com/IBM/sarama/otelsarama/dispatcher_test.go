// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func TestConsumerMessageTextMapCarrierAdapterGetSetKeys(t *testing.T) {
	msg := &sarama.ConsumerMessage{
		Headers: []*sarama.RecordHeader{
			{Key: []byte("key1"), Value: []byte("val1")},
			nil,
			{Key: []byte("key2"), Value: []byte("val2")},
		},
	}
	adapter := &consumerMessageTextMapCarrierAdapter{msg: msg}

	// Get: existing key and missing key.
	assert.Equal(t, "val1", adapter.Get("key1"))
	assert.Empty(t, adapter.Get("missing"))

	// Keys: nil header slots are skipped.
	assert.Equal(t, []string{"key1", "key2"}, adapter.Keys())

	// Set: update existing key in-place (dedup path).
	adapter.Set("key1", "updated")
	assert.Equal(t, "updated", adapter.Get("key1"))
	assert.Len(t, msg.Headers, 3) // nil slot still present, no new entry

	// Set: append a brand-new key.
	adapter.Set("key3", "val3")
	assert.Equal(t, "val3", adapter.Get("key3"))
	assert.Len(t, msg.Headers, 4)
}

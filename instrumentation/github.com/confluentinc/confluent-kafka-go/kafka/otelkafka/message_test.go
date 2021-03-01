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
	"testing"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
)

func TestMessageCarrierSet(t *testing.T) {
	kafkaMsg := &kafka.Message{
		Headers: []kafka.Header{
			{Key: "foo", Value: []byte("bar")},
		},
	}
	carrier := NewMessageCarrier(kafkaMsg)
	carrier.Set("foo", "bar2")
	carrier.Set("foo2", "bar2")
	carrier.Set("foo2", "bar3")
	carrier.Set("foo3", "bar4")
	assert.ElementsMatch(t, carrier.msg.Headers, []kafka.Header{
		{Key: ("foo"), Value: []byte("bar2")},
		{Key: ("foo2"), Value: []byte("bar3")},
		{Key: ("foo3"), Value: []byte("bar4")},
	})
}

func TestMessageCarrierGet(t *testing.T) {
	kafkaMsg := &kafka.Message{}
	testCases := []struct {
		key   string
		value string
	}{
		{"foo", "bar2"},
		{"foo2", "bar2"},
		{"foo3", "bar4"},
	}
	t.Log("Set")
	carrier := NewMessageCarrier(kafkaMsg)
	for _, c := range testCases {
		carrier.Set(c.key, c.value)
	}
	t.Log("Get")
	for _, c := range testCases {
		assert.Equal(t, c.value, carrier.Get(c.key))
	}
}

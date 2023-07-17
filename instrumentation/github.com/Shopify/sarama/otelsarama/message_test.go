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

package otelsarama

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func TestProducerMessageCarrierGet(t *testing.T) {
	testCases := []struct {
		name     string
		carrier  ProducerMessageCarrier
		key      string
		expected string
	}{
		{
			name: "exists",
			carrier: ProducerMessageCarrier{msg: &sarama.ProducerMessage{Headers: []sarama.RecordHeader{
				{Key: []byte("foo"), Value: []byte("bar")},
			}}},
			key:      "foo",
			expected: "bar",
		},
		{
			name:     "not exists",
			carrier:  ProducerMessageCarrier{msg: &sarama.ProducerMessage{Headers: []sarama.RecordHeader{}}},
			key:      "foo",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.carrier.Get(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestProducerMessageCarrierSet(t *testing.T) {
	msg := sarama.ProducerMessage{Headers: []sarama.RecordHeader{
		{Key: []byte("foo"), Value: []byte("bar")},
	}}
	carrier := ProducerMessageCarrier{msg: &msg}

	carrier.Set("foo", "bar2")
	carrier.Set("foo2", "bar2")
	carrier.Set("foo2", "bar3")
	carrier.Set("foo3", "bar4")

	assert.ElementsMatch(t, carrier.msg.Headers, []sarama.RecordHeader{
		{Key: []byte("foo"), Value: []byte("bar2")},
		{Key: []byte("foo2"), Value: []byte("bar3")},
		{Key: []byte("foo3"), Value: []byte("bar4")},
	})
}

func TestProducerMessageCarrierKeys(t *testing.T) {
	testCases := []struct {
		name     string
		carrier  ProducerMessageCarrier
		expected []string
	}{
		{
			name: "one",
			carrier: ProducerMessageCarrier{msg: &sarama.ProducerMessage{Headers: []sarama.RecordHeader{
				{Key: []byte("foo"), Value: []byte("bar")},
			}}},
			expected: []string{"foo"},
		},
		{
			name:     "none",
			carrier:  ProducerMessageCarrier{msg: &sarama.ProducerMessage{Headers: []sarama.RecordHeader{}}},
			expected: []string{},
		},
		{
			name: "many",
			carrier: ProducerMessageCarrier{msg: &sarama.ProducerMessage{Headers: []sarama.RecordHeader{
				{Key: []byte("foo"), Value: []byte("bar")},
				{Key: []byte("baz"), Value: []byte("quux")},
			}}},
			expected: []string{"foo", "baz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.carrier.Keys()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConsumerMessageCarrierGet(t *testing.T) {
	testCases := []struct {
		name     string
		carrier  ConsumerMessageCarrier
		key      string
		expected string
	}{
		{
			name: "exists",
			carrier: ConsumerMessageCarrier{msg: &sarama.ConsumerMessage{Headers: []*sarama.RecordHeader{
				{Key: []byte("foo"), Value: []byte("bar")},
			}}},
			key:      "foo",
			expected: "bar",
		},
		{
			name:     "not exists",
			carrier:  ConsumerMessageCarrier{msg: &sarama.ConsumerMessage{Headers: []*sarama.RecordHeader{}}},
			key:      "foo",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.carrier.Get(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConsumerMessageCarrierSet(t *testing.T) {
	msg := sarama.ConsumerMessage{Headers: []*sarama.RecordHeader{
		{Key: []byte("foo"), Value: []byte("bar")},
	}}
	carrier := ConsumerMessageCarrier{msg: &msg}

	carrier.Set("foo", "bar2")
	carrier.Set("foo2", "bar2")
	carrier.Set("foo2", "bar3")
	carrier.Set("foo3", "bar4")

	assert.ElementsMatch(t, carrier.msg.Headers, []*sarama.RecordHeader{
		{Key: []byte("foo"), Value: []byte("bar2")},
		{Key: []byte("foo2"), Value: []byte("bar3")},
		{Key: []byte("foo3"), Value: []byte("bar4")},
	})
}

func TestConsumerMessageCarrierKeys(t *testing.T) {
	testCases := []struct {
		name     string
		carrier  ConsumerMessageCarrier
		expected []string
	}{
		{
			name: "one",
			carrier: ConsumerMessageCarrier{msg: &sarama.ConsumerMessage{Headers: []*sarama.RecordHeader{
				{Key: []byte("foo"), Value: []byte("bar")},
			}}},
			expected: []string{"foo"},
		},
		{
			name:     "none",
			carrier:  ConsumerMessageCarrier{msg: &sarama.ConsumerMessage{Headers: []*sarama.RecordHeader{}}},
			expected: []string{},
		},
		{
			name: "many",
			carrier: ConsumerMessageCarrier{msg: &sarama.ConsumerMessage{Headers: []*sarama.RecordHeader{
				{Key: []byte("foo"), Value: []byte("bar")},
				{Key: []byte("baz"), Value: []byte("quux")},
			}}},
			expected: []string{"foo", "baz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.carrier.Keys()
			assert.Equal(t, tc.expected, result)
		})
	}
}

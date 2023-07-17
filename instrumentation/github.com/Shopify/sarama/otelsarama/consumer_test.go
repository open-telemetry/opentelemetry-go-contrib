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
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/trace"
)

const (
	topic = "test-topic"
)

func TestConsumerConsumePartitionWithError(t *testing.T) {
	// Mock partition consumer controller
	mockConsumer := mocks.NewConsumer(t, sarama.NewConfig())
	mockConsumer.ExpectConsumePartition(topic, 0, 0)

	consumer := WrapConsumer(mockConsumer)
	_, err := consumer.ConsumePartition(topic, 0, 0)
	assert.NoError(t, err)
	// Consume twice
	_, err = consumer.ConsumePartition(topic, 0, 0)
	assert.Error(t, err)
}

func BenchmarkWrapPartitionConsumer(b *testing.B) {
	// Mock provider
	provider := trace.NewNoopTracerProvider()

	mockPartitionConsumer, partitionConsumer := createMockPartitionConsumer(b)

	partitionConsumer = WrapPartitionConsumer(partitionConsumer, WithTracerProvider(provider))
	message := sarama.ConsumerMessage{Key: []byte("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPartitionConsumer.YieldMessage(&message)
		<-partitionConsumer.Messages()
	}
}

func BenchmarkMockPartitionConsumer(b *testing.B) {
	mockPartitionConsumer, partitionConsumer := createMockPartitionConsumer(b)

	message := sarama.ConsumerMessage{Key: []byte("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPartitionConsumer.YieldMessage(&message)
		<-partitionConsumer.Messages()
	}
}

func createMockPartitionConsumer(b *testing.B) (*mocks.PartitionConsumer, sarama.PartitionConsumer) {
	// Mock partition consumer controller
	consumer := mocks.NewConsumer(b, sarama.NewConfig())
	mockPartitionConsumer := consumer.ExpectConsumePartition(topic, 0, 0)

	// Create partition consumer
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, 0)
	require.NoError(b, err)
	return mockPartitionConsumer, partitionConsumer
}

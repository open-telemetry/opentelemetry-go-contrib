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

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka"
	"go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka/example"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	// Initialize Tracer
	example.InitTracer()

	kafkaConfig := &kafka.ConfigMap{"bootstrap.servers": os.Getenv("KAFKA_PEERS")}
	p, err := otelkafka.NewProducer(
		kafkaConfig, otelkafka.WithContext(context.Background()),
		otelkafka.WithTracerProvider(example.TraceProvider),
		otelkafka.WithPropagators(example.Propagators))
	if err != nil {
		panic(err)
	}
	defer p.Close()

	// Delivery report handler for produced messages
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	// Produce messages to topic (asynchronously)
	topic := "myTopic"
	for _, word := range []string{"Welcome", "to", "the", "Confluent", "Kafka", "Golang", "client"} {
		err := p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(word),
		}, nil)
		if err != nil {
			log.Fatalf("Failed to send messages to kafka. Error:%v", err)
		}
	}

	// Wait for message deliveries before shutting down
	p.Flush(15 * 1000)
}

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

	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":        os.Getenv("KAFKA_PEERS"),
		"group.id":                 "myGroup",
		"auto.offset.reset":        "earliest",
		"go.events.channel.enable": true,
	}
	c, err := otelkafka.NewConsumer(kafkaConfig,
		otelkafka.WithContext(context.Background()),
		otelkafka.WithTracerProvider(example.TraceProvider),
		otelkafka.WithPropagators(example.Propagators),
	)
	if err != nil {
		panic(err)
	}

	err = c.SubscribeTopics([]string{"myTopic", "^aRegex.*[Tt]opic"}, nil)
	if err != nil {
		log.Fatalf("Failed to subribe to topics. Error:%v", err)
	}

	for ev := range c.Events() {
		switch e := ev.(type) {
		case *kafka.Message:
			if e.TopicPartition.Error != nil {
				fmt.Printf("Consumer TopicPartition error: %v (%v)\n", e.TopicPartition.Error, e)
			} else {
				fmt.Printf("Message on %s: %s\n", e.TopicPartition, string(e.Value))
			}

		case kafka.Error:
			fmt.Printf("Consumer error: %v (%v)\n", e.Error(), e)
		default:
		}
	}

	c.Close()
}

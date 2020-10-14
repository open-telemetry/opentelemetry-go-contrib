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
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/semconv"

	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama/example"
)

var (
	brokers = flag.String("brokers", os.Getenv("KAFKA_PEERS"), "The Kafka brokers to connect to, as a comma separated list")
)

func main() {
	example.InitTracer()
	flag.Parse()

	if *brokers == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	brokerList := strings.Split(*brokers, ",")
	log.Printf("Kafka brokers: %s", strings.Join(brokerList, ", "))

	startConsumerGroup(brokerList)

	select {}
}

func startConsumerGroup(brokerList []string) {
	consumerGroupHandler := Consumer{}
	// Wrap instrumentation
	handler := otelsarama.WrapConsumerGroupHandler(&consumerGroupHandler)

	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(brokerList, "example", config)
	if err != nil {
		log.Fatalln("Failed to start sarama consumer group:", err)
	}

	err = consumerGroup.Consume(context.Background(), []string{example.KafkaTopic}, handler)
	if err != nil {
		log.Fatalln("Failed to consume via handler:", err)
	}
}

func printMessage(msg *sarama.ConsumerMessage) {
	// Extract tracing info from message
	ctx := global.TextMapPropagator().Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(msg))

	tr := global.Tracer("consumer")
	_, span := tr.Start(ctx, "consume message", trace.WithAttributes(
		semconv.MessagingOperationProcess,
	))
	defer span.End()

	// Emulate Work loads
	time.Sleep(1 * time.Second)

	log.Println("Successful to read message: ", string(msg.Value))
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		printMessage(message)
		session.MarkMessage(message, "")
	}

	return nil
}

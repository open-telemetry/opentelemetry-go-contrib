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
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/api/global"

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

	producer := newAccessLogProducer(brokerList)

	rand.Seed(time.Now().Unix())

	// Create root span
	tr := global.Tracer("producer")
	ctx, span := tr.Start(context.Background(), "produce message")
	defer span.End()

	// Inject tracing info into message
	msg := sarama.ProducerMessage{
		Topic: example.KafkaTopic,
		Key:   sarama.StringEncoder("random_number"),
		Value: sarama.StringEncoder(fmt.Sprintf("%d", rand.Intn(1000))),
	}
	global.TextMapPropagator().Inject(ctx, otelsarama.NewProducerMessageCarrier(&msg))

	producer.Input() <- &msg
	successMsg := <-producer.Successes()
	log.Println("Successful to write message, offset:", successMsg.Offset)

	err := producer.Close()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Fatalln("Failed to close producer:", err)
	}
}

func newAccessLogProducer(brokerList []string) sarama.AsyncProducer {
	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	// So we can know the partition and offset of messages.
	config.Producer.Return.Successes = true

	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	// Wrap instrumentation
	producer = otelsarama.WrapAsyncProducer(config, producer)

	// We will log to STDOUT if we're not able to produce messages.
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write message:", err)
		}
	}()

	return producer
}

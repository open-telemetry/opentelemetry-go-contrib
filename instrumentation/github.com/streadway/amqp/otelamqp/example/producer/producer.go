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
	"github.com/streadway/amqp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/streadway/amqp/otelamqp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/streadway/amqp/otelamqp/example"
)

func failOnError(err error, msg string) {
	if err != nil {
		fmt.Println(err)
	}
}
func main() {
	example.InitTracer()

	//Make a connection
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")

	defer conn.Close()

	//Ccreate a channel
	ch, _ := conn.Channel()
	defer ch.Close()

	//Declare a queue
	q, _ := ch.QueueDeclare(
		"hello", // name of the queue
		false,   // should the message be persistent? also queue will survive if the cluster gets reset
		false,   // autodelete if there's no consumers (like queues that have anonymous names, often used with fanout exchange)
		false,   // exclusive means I should get an error if any other consumer subsribes to this queue
		false,   // no-wait means I don't want RabbitMQ to wait if there's a queue successfully setup
		nil,     // arguments for more advanced configuration
	)

	//Publish a message
	body := "hello world"
	publishing := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
	}
	span := otelamqp.StartProducerSpan(context.Background(), publishing.Headers)

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		publishing)

	defer otelamqp.EndProducerSpan(span, err)
}

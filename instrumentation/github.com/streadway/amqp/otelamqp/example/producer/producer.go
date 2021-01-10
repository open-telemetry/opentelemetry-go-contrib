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
		fmt.Println( err)
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
	q, err := ch.QueueDeclare(
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
	span := otelamqp.StartProducerSpan(publishing.Headers, context.Background())

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		publishing)

	defer otelamqp.EndProducerSpan(span, err)
}

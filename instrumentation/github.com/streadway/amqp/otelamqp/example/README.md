# Amqp instrumentation example

These instructions expect you have
[docker-compose](https://docs.docker.com/compose/) installed.


Bring up the `RabbitMQ` service to run the
example:

```sh
docker-compose up -d rabbitmq
```

Then up the `rabbitmq-producer` service to publish a message into Rabbitmq:

```sh
docker-compose up rabbitmq-producer
```

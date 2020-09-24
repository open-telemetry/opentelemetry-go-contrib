# Kafka Sarama instrumentation example

A Kafka producer and consumer using Sarama with instrumentation.

These instructions expect you have
[docker-compose](https://docs.docker.com/compose/) installed.

Bring up the `Kafka` and `ZooKeeper` services to run the
example:

```sh
docker-compose up -d zoo kafka
```

Then up the `kafka-producer` service to produce a message into Kafka:

```sh
docker-compose up kafka-producer
```

At last, up the `kafka-consumer` service to consume messages from Kafka:

```sh
docker-compose up kafka-consumer
```

Shut down the services when you are finished with the example:

```sh
docker-compose down
```

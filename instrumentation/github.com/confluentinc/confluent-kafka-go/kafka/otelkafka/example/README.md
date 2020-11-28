# confluent-kafka-go client instrumentation example

A Kafka producer and consumer using confluent-kafka-go package with instrumentation.

These instructions expect you have
[docker-compose](https://docs.docker.com/compose/) installed.

Bring up the `ZooKeeper` and `Broker` services to run the
example:

```sh
docker-compose up -d zookeeper broker
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
docker container prune
```

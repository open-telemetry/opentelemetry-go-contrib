# nats-io/nats.go instrumentation example

These instructions expect you have
[docker-compose](https://docs.docker.com/compose/) installed.

This example will run nats server, producer and consumer.

To start all containers:
```
docker-compose up
```

Producer will publish messages periodically and print TraceID and SpanID.
Consumer will create span from message and print TraceID and SpanID.
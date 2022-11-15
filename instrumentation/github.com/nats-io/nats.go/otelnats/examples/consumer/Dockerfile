FROM golang:alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/nats-io/nats.go/otelnats/examples

FROM base AS otelnats-consumer
RUN go install ./consumer/consumer.go
CMD ["/go/bin/consumer"]
FROM golang:alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/nats-io/nats.go/otelnats/examples

FROM base AS otelnats-producer
RUN go install ./producer/producer.go
CMD ["/go/bin/producer"]
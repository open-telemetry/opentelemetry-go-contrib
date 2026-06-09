# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/otelhttp/example

FROM base AS example-http-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-http-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

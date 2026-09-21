# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/otelhttp/example

FROM base AS example-http-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-http-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

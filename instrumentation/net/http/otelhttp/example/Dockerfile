# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.3-alpine@sha256:d530cf60723abf1ffae0cd16bb546b379ec0550d2f3815e5b16ed69dbc00fba4 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/otelhttp/example

FROM base AS example-http-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-http-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

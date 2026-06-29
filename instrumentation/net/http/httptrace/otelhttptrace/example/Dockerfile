# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/httptrace/otelhttptrace/example

FROM base AS example-httptrace-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-httptrace-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

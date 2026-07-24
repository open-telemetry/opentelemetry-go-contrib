# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/httptrace/otelhttptrace/example

FROM base AS example-httptrace-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-httptrace-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

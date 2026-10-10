# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.2-alpine@sha256:85dc1069ac644ea3c527b177303a406eb3358192816cd7f9e5848eb658851673 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/httptrace/otelhttptrace/example

FROM base AS example-httptrace-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-httptrace-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

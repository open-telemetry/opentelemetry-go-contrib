# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/gorilla/mux/otelmux/example

FROM base AS mux-server
RUN go install ./server.go
CMD ["/go/bin/server"]

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/gin-gonic/gin/otelgin/example

FROM base AS gin-server
RUN go install ./server.go
CMD ["/go/bin/server"]

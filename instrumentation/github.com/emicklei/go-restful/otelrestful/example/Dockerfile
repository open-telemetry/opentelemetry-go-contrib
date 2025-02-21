# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.24-alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/emicklei/go-restful/otelrestful/example

FROM base AS go-restful-server
RUN go install ./server.go
CMD ["/go/bin/server"]

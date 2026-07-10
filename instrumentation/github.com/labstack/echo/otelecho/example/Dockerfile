# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/labstack/echo/otelecho/example

FROM base AS echo-server
RUN go install ./server.go
CMD ["/go/bin/server"]

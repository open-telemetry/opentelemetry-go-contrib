# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/labstack/echo/otelecho/example

FROM base AS echo-server
RUN go install ./server.go
CMD ["/go/bin/server"]

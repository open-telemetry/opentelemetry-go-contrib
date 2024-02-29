# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/gopkg.in/macaron.v1/otelmacaron/example

FROM base AS macaron-server
RUN go install ./server.go
CMD ["/go/bin/server"]

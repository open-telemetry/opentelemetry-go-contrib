# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26-alpine@sha256:f1ddd9fe14fffc091dd98cb4bfa999f32c5fc77d2f2305ea9f0e2595c5437c14 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/httptrace/otelhttptrace/example

FROM base AS example-httptrace-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-httptrace-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

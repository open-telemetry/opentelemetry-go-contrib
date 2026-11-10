# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/otelhttp/example

FROM base AS example-http-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-http-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.1-alpine@sha256:3f6d04dc61331ee3c2fbbaad62d54412a84680f6a041d269a20a5270a078515b AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/httptrace/otelhttptrace/example

FROM base AS example-httptrace-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-httptrace-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

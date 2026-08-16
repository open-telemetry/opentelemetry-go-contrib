# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26-alpine@sha256:70b46548e42db77e0966aaf3619fd068734dc6c77584d526b91126504fd95816 AS base
COPY . /src/
WORKDIR /src/instrumentation/net/http/otelhttp/example

FROM base AS example-http-server
RUN go install ./server/server.go
CMD ["/go/bin/server"]

FROM base AS example-http-client
RUN go install ./client/client.go
CMD ["/go/bin/client"]

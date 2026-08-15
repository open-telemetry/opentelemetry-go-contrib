# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:alpine@sha256:70b46548e42db77e0966aaf3619fd068734dc6c77584d526b91126504fd95816 AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/labstack/echo/otelecho/example

FROM base AS echo-server
RUN go install ./server.go
CMD ["/go/bin/server"]

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.2-alpine@sha256:85dc1069ac644ea3c527b177303a406eb3358192816cd7f9e5848eb658851673 AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

FROM base AS aws-client
RUN go install ./main.go
CMD ["/go/bin/main"]

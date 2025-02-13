# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.24-alpine AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

FROM base AS aws-client
RUN go install ./main.go
CMD ["/go/bin/main"]

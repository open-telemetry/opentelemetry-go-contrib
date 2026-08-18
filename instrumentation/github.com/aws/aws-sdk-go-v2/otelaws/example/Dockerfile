# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

FROM base AS aws-client
RUN go install ./main.go
CMD ["/go/bin/main"]

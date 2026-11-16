# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

FROM base AS aws-client
RUN go install ./main.go
CMD ["/go/bin/main"]

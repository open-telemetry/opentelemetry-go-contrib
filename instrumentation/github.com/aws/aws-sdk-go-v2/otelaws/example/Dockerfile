# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/example

FROM base AS aws-client
RUN go install ./main.go
CMD ["/go/bin/main"]

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.1@sha256:3680233e3204827fbdc66088528ae6d4b3d034f51d03a99d454f6de034888244 AS base
COPY . /src/
WORKDIR /src/instrumentation/github.com/aws/aws-lambda-go/otellambda/example
RUN apt-get update

FROM base AS aws-lambda

# install other package(s) in base
RUN apt-get install zip unzip
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip \
    && ./aws/install
RUN apt-get -y install jq

CMD ["./build.sh"]

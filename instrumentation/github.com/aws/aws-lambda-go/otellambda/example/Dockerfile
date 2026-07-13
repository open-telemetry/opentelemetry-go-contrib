# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26@sha256:079e59808d2d252516e27e3f3a9c003740dee7f75e55aa71528766d52bcfc16a AS base
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

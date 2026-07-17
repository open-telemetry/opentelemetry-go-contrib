# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26@sha256:983a0823d3dab83604654972fe6bbda13142a7c57f987804fbdddb9d47dad9ec AS base
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

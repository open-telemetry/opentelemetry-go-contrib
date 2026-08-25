# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.0@sha256:65b6f280bf050ec5af12716857e8ea8439d694dbba8f31ceeb7630670071f2bb AS base
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

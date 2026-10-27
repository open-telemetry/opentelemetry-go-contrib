# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.3@sha256:9048d5e7751706200eb1719c7dae0436950d362f29a1da91fa05e5c63867c60f AS base
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

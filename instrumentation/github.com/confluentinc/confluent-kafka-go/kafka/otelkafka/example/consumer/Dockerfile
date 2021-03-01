# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
FROM golang:alpine AS base
RUN mkdir -p /app/bin
WORKDIR /app
COPY . /app/
ENV GOBIN=$PWD/app/bin
WORKDIR /app/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka/example

FROM base AS kafka-consumer
RUN apk add build-base
RUN go install -tags musl ./consumer/consumer.go
CMD ["app/bin/consumer"]

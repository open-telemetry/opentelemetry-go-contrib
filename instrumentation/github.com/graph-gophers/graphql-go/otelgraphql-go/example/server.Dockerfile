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
COPY . /src/
WORKDIR /src/instrumentation/github.com/graph-gophers/graphql-go/otelgraphql-go/example

FROM base AS graphql-go-server
RUN go install ./server.go
CMD ["/go/bin/server"]

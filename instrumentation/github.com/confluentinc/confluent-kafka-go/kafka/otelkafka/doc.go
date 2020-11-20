// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package otelkafka instruments the github.com/confluentinc/confluent-kafka-go package.
//
// The consumer's span will be created as a child of the producer's span.
//
// Based on: https://github.com/DataDog/dd-trace-go/tree/v1/contrib/confluentinc/confluent-kafka-go/kafka
package otelkafka // import "go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka"

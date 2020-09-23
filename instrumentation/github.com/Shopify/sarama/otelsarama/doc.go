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

// Package otelsarama instruments the github.com/Shopify/sarama package.
//
// The consumer's span will be created as a child of the producer's span.
//
// Context propagation only works on Kafka versions higher than 0.11.0.0 which supports record headers.
// (https://archive.apache.org/dist/kafka/0.11.0.0/RELEASE_NOTES.html)
//
// Based on: https://github.com/DataDog/dd-trace-go/tree/v1/contrib/Shopify/sarama
package otelsarama // import "go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"

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

// Package sarama provides functions to trace the Shopify/sarama package. (https://github.com/Shopify/sarama)
//
// The consumer's span will not be created as a child of the producer's span; instead, it will link the producer's span.
// (https://github.com/open-telemetry/opentelemetry-specification/blob/v0.6.0/specification/trace/semantic_conventions/messaging.md#batch-receiving)
//
// Context propagation only works on Kafka versions higher than 0.11.0.0 which supports record headers.
// (https://archive.apache.org/dist/kafka/0.11.0.0/RELEASE_NOTES.html)
//
// Based on: https://github.com/DataDog/dd-trace-go/tree/v1/contrib/Shopify/sarama
package sarama // import "go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama"

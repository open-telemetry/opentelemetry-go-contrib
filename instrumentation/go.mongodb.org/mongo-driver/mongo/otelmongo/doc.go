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

// Package otelmongo instruments go.mongodb.org/mongo-driver/mongo.
//
// This package is compatable with v0.2.0 of
// go.mongodb.org/mongo-driver/mongo.
//
// `NewMonitor` will return an event.CommandMonitor which is used to trace
// requests.
//
// This code was originally based on the following:
// - https://github.com/DataDog/dd-trace-go/tree/02f0449efa3cb382d499fadc873957385dcb2192/contrib/go.mongodb.org/mongo-driver/mongo
// - https://github.com/DataDog/dd-trace-go/tree/v1.23.3/ddtrace/ext
package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

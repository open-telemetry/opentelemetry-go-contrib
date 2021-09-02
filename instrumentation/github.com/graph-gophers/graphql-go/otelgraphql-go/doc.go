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

// Package otelgraphqlgo instruments the graph-gophers/graphql-go package
// (https://github.com/graph-gophers/graphql-go).
//
// graphql-go already provides a tracer interface along with its respective
// opentracing implementation
// (https://github.com/graph-gophers/graphql-go/tree/v1.1.0/trace)
// This interface consists of the following methods, all of which get context as
// a parameter.
//
// TraceValidation: traces the schema validation step which precedes the actual
// operation.
//
// TraceQuery: traces the actual operation, query or mutation, as a whole
//
// TraceField: traces a field-specific operation; its span should typically be a
// sub-span of the TraceQuery one;
//
// otelgraphqlgo provides an implementation of this interface which is
// practically a port of the opentracing one that comes with graphql-go.
//
// Some other points:
//
// a. graphql-go exposes a single HTTP Handler for all graphql operations. That
// makes it a natural fit for otelhttp and other router packages
// instrumentation, if propagating frontend baggage (e.g. ) is required.
//
// b. graphql-go resolver methods do get context as a parameter, which allows
// for field-specific (sub-)span creation, if TraceQuery does not suffice.
package otelgraphqlgo // import "go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo"

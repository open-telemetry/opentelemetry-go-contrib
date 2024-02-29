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

// Package otelhttp provides an http.Handler and functions that are intended
// to be used to add tracing by wrapping existing handlers (with Handler) and
// routes WithRouteTag.
//
// Warning: migration of semantic conventions to v1.24.0 is in progress. Because
// this will break most existing dashboards we have developed a migration plan
// detailed [here](). Use the environment variable `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE`
// to opt into the new conventions. This will be removed in a future release.
package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

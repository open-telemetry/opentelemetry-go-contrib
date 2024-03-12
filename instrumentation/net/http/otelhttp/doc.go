// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelhttp provides an http.Handler and functions that are intended
// to be used to add tracing by wrapping existing handlers (with Handler) and
// routes WithRouteTag.
//
// Warning: migration of semantic conventions to v1.24.0 is in progress. Because
// this will break most existing dashboards we have developed a migration plan
// detailed [here](). Use the environment variable `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE`
// to opt into the new conventions. This will be removed in a future release.
package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

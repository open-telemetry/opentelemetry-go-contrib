// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelhttp provides an http.Handler and functions that are intended
// to be used to add tracing by wrapping existing handlers.
// Use [WithRouteTracking] on the instrumentation and [RouteMiddleware] directly
// around an [http.ServeMux] to preserve route telemetry when middleware between
// the instrumentation and mux copies requests.
package otelhttp

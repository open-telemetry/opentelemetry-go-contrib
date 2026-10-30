// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package zpages implements a collection of in-process HTML pages that display
// status and live diagnostic telemetry from a running application.
//
// The tracez page provides an HTTP handler that displays summaries of active spans
// as well as latency-bucketed and error-bucketed samples of completed spans collected
// by [SpanProcessor].
//
// To use zpages:
//
//  1. Create a [SpanProcessor] with [NewSpanProcessor] and register it with the
//     OpenTelemetry SDK [go.opentelemetry.io/otel/sdk/trace.TracerProvider].
//  2. Register the handler returned by [NewTracezHandler] on an HTTP server mux
//     (for example, at "/debug/tracez").
//
// Users can then open the endpoint in a browser to inspect span diagnostics in real time.
package zpages

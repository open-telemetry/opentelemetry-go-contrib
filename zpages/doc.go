// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package zpages implements in-process HTML pages that display live
// diagnostic trace telemetry from a running application.
//
// The tracez page provides an HTTP handler ([NewTracezHandler]) that renders
// a summary table of active spans along with latency-bucketed and
// error-bucketed samples of completed spans collected by [SpanProcessor].
// Selecting a count in the summary table displays details for the matching
// spans, including trace and span IDs, timestamps, durations, status,
// attributes, and events.
//
// # Usage
//
// To use zpages:
//
//  1. Create a [SpanProcessor] with [NewSpanProcessor] and register it with an
//     OpenTelemetry SDK [go.opentelemetry.io/otel/sdk/trace.TracerProvider].
//  2. Register the handler returned by [NewTracezHandler] on a dedicated
//     administrative HTTP listener (for example, bound to loopback at
//     "/debug/tracez"), or wrap the handler with authentication and
//     authorization middleware.
//
// # Access Controls
//
// [NewTracezHandler] renders span names, trace and span IDs, status
// descriptions, span attributes, and span events (including event attributes)
// directly in HTML without built-in authentication, authorization, or
// redaction. Because spans can contain sensitive request metadata, user data,
// or error details, do not expose the handler on a public or unauthenticated
// listener.
//
// Before using zpages in production, restrict access by binding the handler to
// a loopback or private administrative listener protected by network controls,
// wrapping it with authentication and authorization middleware, or both.
//
// # Span Retention
//
// [SpanProcessor] keeps span diagnostics in process memory:
//
//   - Active spans: Every recording span passed to [SpanProcessor.OnStart]
//     with a valid [go.opentelemetry.io/otel/trace.SpanContext] is stored and
//     removed on [SpanProcessor.OnEnd] (spans dropped by the sampler are
//     non-recording and are not passed to the processor). Active span storage
//     is not currently bounded by count, so recording spans that are never
//     ended remain in memory for the lifetime of the processor.
//   - Completed span samples: On [SpanProcessor.OnEnd], spans are grouped by
//     span name. Spans with status code [go.opentelemetry.io/otel/codes.Error]
//     are sampled into a per-name error bucket; all other completed spans are
//     sampled into 9 latency buckets with lower bounds of 0s, 10µs, 100µs,
//     1ms, 10ms, 100ms, 1s, 10s, and 100s (negative durations are treated as 0).
//   - Per-bucket limits: Each latency bucket and error bucket holds up to 10
//     spans in a circular buffer (overwriting the oldest entry when full) and
//     accepts at most one span per second based on span end time.
//   - Distinct span names: The number of distinct span names retained in
//     completed sample storage is not currently bounded, and span names are
//     never evicted. Avoid unbounded or caller-controlled span names, as each
//     distinct name allocates and retains its own set of buckets.
//
// # Shutdown Behavior
//
// [SpanProcessor.ForceFlush] is a no-op because diagnostics are stored in
// memory rather than exported.
//
// [SpanProcessor.Shutdown] is currently a no-op and returns nil without
// clearing active spans or completed span samples. When the parent
// [go.opentelemetry.io/otel/sdk/trace.TracerProvider] shuts down, the SDK
// stops invoking [SpanProcessor.OnStart] and [SpanProcessor.OnEnd]. If an HTTP
// handler backed by the processor continues serving after provider shutdown,
// previously sampled spans remain visible and any spans that were active at
// shutdown remain listed as active indefinitely. Shut down the HTTP server
// (or stop routing to and drop references to the handler and SpanProcessor;
// note that [net/http.ServeMux] cannot unregister routes) before shutting down
// the TracerProvider so requests do not observe stale state and retained spans
// can be reclaimed.
package zpages

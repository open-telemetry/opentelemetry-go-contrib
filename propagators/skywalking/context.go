// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package skywalking // import "go.opentelemetry.io/contrib/propagators/skywalking"

import "context"

type skyWalkingKeyType int

const (
	tracingModeKey skyWalkingKeyType = iota
	timestampKey
)

// Tracing mode constants for SW8-X extension header.
const (
	// TracingModeNormal indicates normal tracing analysis (default).
	TracingModeNormal = "0"
	// TracingModeSkipAnalysis indicates the trace should skip analysis.
	TracingModeSkipAnalysis = "1"
)

// withTracingMode returns a copy of parent with the tracing mode set.
//
// The tracing mode is used in the SW8-X extension header to control
// how SkyWalking handles trace analysis:
// - TracingModeNormal ("0"): Normal analysis (default)
// - TracingModeSkipAnalysis ("1"): Skip analysis
func withTracingMode(parent context.Context, mode string) context.Context {
	return context.WithValue(parent, tracingModeKey, mode)
}

// tracingModeFromContext returns the tracing mode stored in ctx.
//
// If no tracing mode is stored in ctx, TracingModeNormal is returned.
func tracingModeFromContext(ctx context.Context) string {
	if ctx == nil {
		return TracingModeNormal
	}
	if mode, ok := ctx.Value(tracingModeKey).(string); ok {
		return mode
	}
	return TracingModeNormal
}

// withTimestamp returns a copy of parent with the timestamp set.
//
// The timestamp is used in the SW8-X extension header for transmission
// latency calculation. It should be set to the current time in milliseconds
// when sending a request to enable latency measurement.
func withTimestamp(parent context.Context, timestamp int64) context.Context {
	return context.WithValue(parent, timestampKey, timestamp)
}

// timestampFromContext returns the timestamp stored in ctx.
//
// If no timestamp is stored in ctx, 0 is returned indicating no timestamp.
func timestampFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if timestamp, ok := ctx.Value(timestampKey).(int64); ok {
		return timestamp
	}
	return 0
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"time"
)

type startTimeContextKeyType int

const startTimeContextKey startTimeContextKeyType = 0

// ContextWithStartTime returns a new context with the provided start time.
// The start time will be used for metrics and traces emitted by the
// instrumentation. Injecting it multiple times will override the previous
// start time.
func ContextWithStartTime(parent context.Context, start time.Time) context.Context {
	return context.WithValue(parent, startTimeContextKey, start)
}

// StartTimeFromContext retrieves a time.Time from the provided context.
// If no start time is found, a zero time is returned.
func StartTimeFromContext(ctx context.Context) time.Time {
	t, _ := ctx.Value(startTimeContextKey).(time.Time)
	return t
}

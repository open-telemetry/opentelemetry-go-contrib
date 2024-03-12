// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"time"
)

// clock represents a time keeper that returns its version of the current time.
type clock interface {
	now() time.Time
}

// defaultClock wraps the standard time package.
type defaultClock struct{}

// now returns current time according to the standard time package.
func (t *defaultClock) now() time.Time {
	return time.Now()
}

// mockClock is a time keeper that returns a fixed time.
type mockClock struct {
	nowTime  int64
	nowNanos int64
}

// now function returns the fixed time value stored in c.
func (c *mockClock) now() time.Time {
	return time.Unix(c.nowTime, c.nowNanos)
}

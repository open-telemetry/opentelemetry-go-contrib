// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package skywalking

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	WithTracingMode        = withTracingMode
	TracingModeFromContext = tracingModeFromContext
	WithTimestamp          = withTimestamp
	TimestampFromContext   = timestampFromContext
)

func TestWithTracingMode(t *testing.T) {
	testCases := []struct {
		name string
		mode string
	}{
		{
			name: "normal mode",
			mode: TracingModeNormal,
		},
		{
			name: "skip analysis mode",
			mode: TracingModeSkipAnalysis,
		},
		{
			name: "custom mode",
			mode: "custom",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := withTracingMode(context.Background(), tc.mode)
			mode := tracingModeFromContext(ctx)
			assert.Equal(t, tc.mode, mode)
		})
	}
}

func TestTracingModeFromContext_Default(t *testing.T) {
	testCases := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "nil context",
			ctx:  nil,
		},
		{
			name: "empty context",
			ctx:  context.Background(),
		},
		{
			name: "context with different value type",
			ctx:  context.WithValue(context.Background(), tracingModeKey, 123),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode := tracingModeFromContext(tc.ctx)
			assert.Equal(t, TracingModeNormal, mode)
		})
	}
}

func TestTracingModeConstants(t *testing.T) {
	assert.Equal(t, "0", TracingModeNormal)
	assert.Equal(t, "1", TracingModeSkipAnalysis)
}

func TestWithTimestamp(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp int64
	}{
		{
			name:      "zero timestamp",
			timestamp: 0,
		},
		{
			name:      "positive timestamp",
			timestamp: 1602743904804,
		},
		{
			name:      "current time",
			timestamp: 1640995200000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := withTimestamp(context.Background(), tc.timestamp)
			timestamp := timestampFromContext(ctx)
			assert.Equal(t, tc.timestamp, timestamp)
		})
	}
}

func TestTimestampFromContext_Default(t *testing.T) {
	testCases := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "nil context",
			ctx:  nil,
		},
		{
			name: "empty context",
			ctx:  context.Background(),
		},
		{
			name: "context with different value type",
			ctx:  context.WithValue(context.Background(), timestampKey, "not-an-int64"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			timestamp := timestampFromContext(tc.ctx)
			assert.Equal(t, int64(0), timestamp)
		})
	}
}

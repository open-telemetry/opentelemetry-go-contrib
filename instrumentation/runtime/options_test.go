// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	for _, tt := range []struct {
		name   string
		opts   []Option
		expect config
	}{
		{
			name:   "default",
			expect: config{MinimumReadMemStatsInterval: 15 * time.Second},
		},
		{
			name:   "negative MinimumReadMemStatsInterval ignored",
			opts:   []Option{WithMinimumReadMemStatsInterval(-1 * time.Second)},
			expect: config{MinimumReadMemStatsInterval: 15 * time.Second},
		},
		{
			name:   "set MinimumReadMemStatsInterval",
			opts:   []Option{WithMinimumReadMemStatsInterval(10 * time.Second)},
			expect: config{MinimumReadMemStatsInterval: 10 * time.Second},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := newConfig(tt.opts...)
			assert.True(t, configEqual(got, tt.expect))
		})
	}
}

func TestOptInMetricEnabled(t *testing.T) {
	assert.False(t, optInGCCycles.enabled())
	assert.False(t, optInCPUTime.enabled())

	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.cpu.time")
	assert.False(t, optInGCCycles.enabled())
	assert.True(t, optInCPUTime.enabled())
}

func configEqual(a, b config) bool {
	// ignore MeterProvider
	return a.MinimumReadMemStatsInterval == b.MinimumReadMemStatsInterval
}

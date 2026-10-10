// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/semconv/v1.43.0/goconv"
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

func TestOptInEnabled(t *testing.T) {
	c := newConfig(WithOptInMetrics(MemoryGCCycles))
	assert.True(t, c.optInEnabled(MemoryGCCycles))
	assert.False(t, c.optInEnabled(CPUTime))

	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.cpu.time")
	assert.True(t, c.optInEnabled(MemoryGCCycles))
	assert.True(t, c.optInEnabled(CPUTime))

	p := newProducerConfig(WithOptInMetrics(MemoryGCPauseDuration))
	assert.True(t, p.optInEnabled(MemoryGCPauseDuration))
}

func TestOptInMetricString(t *testing.T) {
	assert.Equal(t, goconv.MemoryGCCyclesObservable{}.Name(), MemoryGCCycles.String())
	assert.Equal(t, goconv.CPUTimeObservable{}.Name(), CPUTime.String())
	assert.Equal(t, goconv.MemoryGCPauseDuration{}.Name(), MemoryGCPauseDuration.String())
}

func configEqual(a, b config) bool {
	// ignore MeterProvider
	return a.MinimumReadMemStatsInterval == b.MinimumReadMemStatsInterval
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus // import "go.opentelemetry.io/contrib/bridges/prometheus"

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	otherRegistry := prometheus.NewRegistry()

	testCases := []struct {
		name       string
		options    []Option
		wantConfig config
	}{
		{
			name:    "Default",
			options: nil,
			wantConfig: config{
				gatherers: []prometheus.Gatherer{prometheus.DefaultGatherer},
			},
		},
		{
			name:    "With a different gatherer",
			options: []Option{WithGatherer(otherRegistry)},
			wantConfig: config{
				gatherers: []prometheus.Gatherer{otherRegistry},
			},
		},
		{
			name:    "Multiple gatherers",
			options: []Option{WithGatherer(otherRegistry), WithGatherer(prometheus.DefaultGatherer)},
			wantConfig: config{
				gatherers: []prometheus.Gatherer{otherRegistry, prometheus.DefaultGatherer},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfig(tt.options...)
			assert.Equal(t, tt.wantConfig, cfg)
		})
	}
}

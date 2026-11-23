// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptInMetrics(t *testing.T) {
	const key = "OTEL_GO_X_RUNTIME_METRICS_OPTIN"
	require.Equal(t, key, OptInMetrics.Key())

	setenv := func(v string) func(t *testing.T) {
		return func(t *testing.T) { t.Setenv(key, v) }
	}

	t.Run("listed", run(setenv("go.memory.gc.cycles,go.cpu.time"), assertListed(OptInMetrics, "go.cpu.time", true)))
	t.Run("spaces", run(setenv(" go.memory.gc.cycles , go.cpu.time "), assertListed(OptInMetrics, "go.memory.gc.cycles", true)))
	t.Run("unlisted", run(setenv("go.memory.gc.cycles"), assertListed(OptInMetrics, "go.cpu.time", false)))
	t.Run("prefix", run(setenv("go.memory.gc"), assertListed(OptInMetrics, "go.memory.gc.cycles", false)))
	t.Run("empty", run(assertListed(OptInMetrics, "go.cpu.time", false)))
}

func assertListed(f ListFeature, value string, listed bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		assert.Equal(t, listed, f.Enabled(value), "listed")
	}
}

func run(steps ...func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		for _, step := range steps {
			step(t)
		}
	}
}

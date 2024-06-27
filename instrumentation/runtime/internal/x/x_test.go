// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeprecatedRuntimeMetrics(t *testing.T) {
	const key = "OTEL_GO_X_DEPRECATED_RUNTIME_METRICS"
	require.Equal(t, key, DeprecatedRuntimeMetrics.Key())

	t.Run("true", run(setenv(key, "true"), assertEnabled(DeprecatedRuntimeMetrics, true)))
	t.Run("True", run(setenv(key, "True"), assertEnabled(DeprecatedRuntimeMetrics, true)))
	t.Run("TRUE", run(setenv(key, "TRUE"), assertEnabled(DeprecatedRuntimeMetrics, true)))
	t.Run("false", run(setenv(key, "false"), assertEnabled(DeprecatedRuntimeMetrics, false)))
	t.Run("False", run(setenv(key, "False"), assertEnabled(DeprecatedRuntimeMetrics, false)))
	t.Run("FALSE", run(setenv(key, "FALSE"), assertEnabled(DeprecatedRuntimeMetrics, false)))
	t.Run("1", run(setenv(key, "1"), assertEnabled(DeprecatedRuntimeMetrics, true)))
	t.Run("empty", run(assertEnabled(DeprecatedRuntimeMetrics, true)))
}

func run(steps ...func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		for _, step := range steps {
			step(t)
		}
	}
}

func setenv(k, v string) func(t *testing.T) { //nolint:unparam
	return func(t *testing.T) { t.Setenv(k, v) }
}

func assertEnabled(f BoolFeature, enabled bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		assert.Equal(t, enabled, f.Enabled(), "not enabled")
	}
}

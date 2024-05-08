// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

func TestNewCoreConfiguration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		r := logtest.NewRecorder()
		prev := global.GetLoggerProvider()
		defer global.SetLoggerProvider(prev)
		global.SetLoggerProvider(r)

		var h *Core
		require.NotPanics(t, func() { h = NewCore() })
		require.NotNil(t, h.logger)
		require.IsType(t, &logtest.Recorder{}, h.logger)
		l := h.logger.(*logtest.Recorder)
		require.Len(t, l.Result(), 1)

		want := &logtest.ScopeRecords{Name: bridgeName, Version: version}
		got := l.Result()[0]
		assert.Equal(t, want, got)
	})

	t.Run("Options", func(t *testing.T) {
		r := logtest.NewRecorder()
		scope := instrumentation.Scope{Name: "name", Version: "ver", SchemaURL: "url"}
		var h *Core
		require.NotPanics(t, func() {
			h = NewCore(
				WithLoggerProvider(r),
				WithInstrumentationScope(scope),
			)
		})
		require.NotNil(t, h.logger)
		require.IsType(t, &logtest.Recorder{}, h.logger)
		l := h.logger.(*logtest.Recorder)
		require.Len(t, l.Result(), 1)

		want := &logtest.ScopeRecords{Name: scope.Name, Version: scope.Version, SchemaURL: scope.SchemaURL}
		got := l.Result()[0]
		assert.Equal(t, want, got)
	})
}

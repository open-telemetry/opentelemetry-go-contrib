// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.
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
		global.SetLoggerProvider(r)

		var h *Core
		assert.NotPanics(t, func() { h = NewCore() })
		assert.NotNil(t, h.logger)
		require.IsType(t, &logtest.Recorder{}, h.logger)

		l := h.logger.(*logtest.Recorder)
		want := &logtest.ScopeRecords{Name: bridgeName, Version: version}
		assert.Equal(t, want, l.Result()[0])
	})

	t.Run("Options", func(t *testing.T) {
		r := logtest.NewRecorder()
		scope := instrumentation.Scope{Name: "name", Version: "ver", SchemaURL: "url"}
		var h *Core
		assert.NotPanics(t, func() {
			h = NewCore(
				WithLoggerProvider(r),
				WithInstrumentationScope(scope),
			)
		})
		assert.NotNil(t, h.logger)
		require.IsType(t, &logtest.Recorder{}, h.logger)

		l := h.logger.(*logtest.Recorder)
		assert.Equal(t, scope.Name, l.Result()[0].Name)
		assert.Equal(t, scope.Version, l.Result()[0].Version)
		assert.Equal(t, scope.SchemaURL, l.Result()[0].SchemaURL)
	})
}

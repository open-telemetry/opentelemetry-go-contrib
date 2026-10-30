// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package diaglogger provides a diagnostic logger configured by severity level.
package diaglogger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		wantInfo    bool
		wantWarning bool
		wantError   bool
	}{
		{
			name:        "trace",
			level:       "trace",
			wantInfo:    true,
			wantWarning: true,
			wantError:   true,
		},
		{
			name:        "debug",
			level:       "debug",
			wantInfo:    true,
			wantWarning: true,
			wantError:   true,
		},
		{
			name:        "info",
			level:       "info",
			wantInfo:    true,
			wantWarning: true,
			wantError:   true,
		},
		{
			name:        "warn",
			level:       "warn",
			wantWarning: true,
			wantError:   true,
		},
		{
			name:      "error",
			level:     "error",
			wantError: true,
		},
		{
			name:  "fatal",
			level: "fatal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer

			logger, err := New(tt.level, &output)
			require.NoError(t, err)

			logger.V(8).Info("debug marker")
			assert.Equal(t, tt.level == "trace" || tt.level == "debug",
				strings.Contains(output.String(), "debug marker"),
				"debug message filtering; output: %q", output.String())

			logger.V(4).Info("info marker")
			logger.V(1).Info("warning marker")
			logger.Error(errors.New("test error"), "error marker")

			got := output.String()

			assert.Equal(t, tt.wantInfo, strings.Contains(got, "info marker"),
				"informational message filtering; output: %q", got)
			assert.Equal(t, tt.wantWarning, strings.Contains(got, "warning marker"),
				"warning message filtering; output: %q", got)
			assert.Equal(t, tt.wantError, strings.Contains(got, "error marker"),
				"error message filtering; output: %q", got)
		})
	}
}

func TestNewWritesToProvidedWriter(t *testing.T) {
	var output bytes.Buffer

	logger, err := New("info", &output)
	require.NoError(t, err)

	logger.Info("capture marker")

	assert.Contains(t, output.String(), "capture marker")
}

func TestNewUnsupportedLevel(t *testing.T) {
	_, err := New("invalid")
	require.EqualError(t, err, `unsupported log_level "invalid"`)
}

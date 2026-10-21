// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diaglogger

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		verbosity int
		wantErr   bool
	}{
		{name: "empty", level: "", wantErr: true},

		{name: "trace", level: "trace", verbosity: 8},
		{name: "trace2", level: "trace2", verbosity: 8},
		{name: "trace3", level: "trace3", verbosity: 8},
		{name: "trace4", level: "trace4", verbosity: 8},

		{name: "debug", level: "debug", verbosity: 8},
		{name: "debug2", level: "debug2", verbosity: 8},
		{name: "debug3", level: "debug3", verbosity: 8},
		{name: "debug4", level: "debug4", verbosity: 8},

		{name: "info", level: "info", verbosity: 4},
		{name: "info2", level: "info2", verbosity: 4},
		{name: "info3", level: "info3", verbosity: 4},
		{name: "info4", level: "info4", verbosity: 4},

		{name: "warn", level: "warn", verbosity: 1},
		{name: "warn2", level: "warn2", verbosity: 1},
		{name: "warn3", level: "warn3", verbosity: 1},
		{name: "warn4", level: "warn4", verbosity: 1},

		{name: "error", level: "error", verbosity: 0},
		{name: "error2", level: "error2", verbosity: 0},
		{name: "error3", level: "error3", verbosity: 0},
		{name: "error4", level: "error4", verbosity: 0},

		{name: "fatal", level: "fatal", verbosity: 0},
		{name: "fatal2", level: "fatal2", verbosity: 0},
		{name: "fatal3", level: "fatal3", verbosity: 0},
		{name: "fatal4", level: "fatal4", verbosity: 0},

		{name: "invalid", level: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.level)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
			if err != nil {
				return
			}

			for _, v := range []int{0, 1, 4, 8, 9} {
				wantEnabled := v <= tt.verbosity
				if got := logger.V(v).Enabled(); got != wantEnabled {
					t.Errorf("New(%q): V(%d).Enabled() = %v, want %v",
						tt.level, v, got, wantEnabled)
				}
			}

			if !logger.Enabled() {
				t.Error("logger should enable non-verbose error messages")
			}
		})
	}
}

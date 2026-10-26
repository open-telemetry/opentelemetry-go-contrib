// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package diaglogger provides a diagnostic logger configured by severity level.
package diaglogger

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
)

// New creates a logger whose verbosity matches the configured OpenTelemetry
// severity. Unsupported or empty levels return an error.
func New(level string) (logr.Logger, error) {
	base := level
	if base != "" {
		last := base[len(base)-1]
		if last >= '2' && last <= '4' {
			base = base[:len(base)-1]
		}
	}

	var verbosity int
	switch base {
	case "trace", "debug":
		verbosity = 8
	case "info":
		verbosity = 4
	case "warn":
		verbosity = 1
	case "error":
		verbosity = 0
	case "fatal":
		return logr.Discard(), nil
	default:
		return logr.Logger{}, fmt.Errorf("unsupported log_level %q", level)
	}

	logger := funcr.NewJSON(func(obj string) {
		_, _ = fmt.Fprintln(os.Stderr, obj)
	}, funcr.Options{Verbosity: verbosity})

	return logger, nil
}

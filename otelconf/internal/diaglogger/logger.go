// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package diaglogger provides a diagnostic logger configured by severity level.
package diaglogger

import (
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
)

// New creates a logger whose verbosity matches the configured OpenTelemetry
// severity. Unsupported or empty levels return an error. If an output writer
// is provided, log messages are written to it; otherwise, os.Stderr is used.
func New(level string, outputs ...io.Writer) (logr.Logger, error) {
	output := io.Writer(os.Stderr)
	if len(outputs) > 0 && outputs[0] != nil {
		output = outputs[0]
	}

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
		_, _ = fmt.Fprintln(output, obj)
	}, funcr.Options{Verbosity: verbosity})

	return logger, nil
}

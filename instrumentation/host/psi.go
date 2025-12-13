// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package host // import "go.opentelemetry.io/contrib/instrumentation/host"

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/metric"
)

const (
	psiCPUFile    = "/proc/pressure/cpu"
	psiMemoryFile = "/proc/pressure/memory"
	psiIOFile     = "/proc/pressure/io"
)

// psiStats represents parsed PSI statistics for a resource
type psiStats struct {
	some psiStat
	full psiStat
}

type psiStat struct {
	avg10  float64
	avg60  float64
	avg300 float64
	total  int64
}

// psiMetrics holds all PSI metric instruments
type psiMetrics struct {
	cpuSomeAvg10     metric.Float64ObservableGauge
	cpuSomeAvg60     metric.Float64ObservableGauge
	cpuSomeAvg300    metric.Float64ObservableGauge
	cpuSomeTotal     metric.Int64ObservableCounter
	memorySomeAvg10  metric.Float64ObservableGauge
	memorySomeAvg60  metric.Float64ObservableGauge
	memorySomeAvg300 metric.Float64ObservableGauge
	memorySomeTotal  metric.Int64ObservableCounter
	memoryFullAvg10  metric.Float64ObservableGauge
	memoryFullAvg60  metric.Float64ObservableGauge
	memoryFullAvg300 metric.Float64ObservableGauge
	memoryFullTotal  metric.Int64ObservableCounter
	ioSomeAvg10      metric.Float64ObservableGauge
	ioSomeAvg60      metric.Float64ObservableGauge
	ioSomeAvg300     metric.Float64ObservableGauge
	ioSomeTotal      metric.Int64ObservableCounter
	ioFullAvg10      metric.Float64ObservableGauge
	ioFullAvg60      metric.Float64ObservableGauge
	ioFullAvg300     metric.Float64ObservableGauge
	ioFullTotal      metric.Int64ObservableCounter
}

// registerPSI registers all PSI metric instruments and their callback
func (h *host) registerPSI() (*psiMetrics, error) {
	pm := &psiMetrics{}
	var err error

	// CPU PSI metrics
	pm.cpuSomeAvg10, err = h.meter.Float64ObservableGauge(
		"system.psi.cpu.some.avg10",
		metric.WithDescription("CPU pressure stall information - some tasks waiting, 10 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.cpuSomeAvg60, err = h.meter.Float64ObservableGauge(
		"system.psi.cpu.some.avg60",
		metric.WithDescription("CPU pressure stall information - some tasks waiting, 60 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.cpuSomeAvg300, err = h.meter.Float64ObservableGauge(
		"system.psi.cpu.some.avg300",
		metric.WithDescription("CPU pressure stall information - some tasks waiting, 300 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.cpuSomeTotal, err = h.meter.Int64ObservableCounter(
		"system.psi.cpu.some.total",
		metric.WithDescription("CPU pressure stall information - some tasks waiting, total time in microseconds"),
		metric.WithUnit("us"),
	)
	if err != nil {
		return nil, err
	}

	// Memory PSI metrics - some
	pm.memorySomeAvg10, err = h.meter.Float64ObservableGauge(
		"system.psi.memory.some.avg10",
		metric.WithDescription("Memory pressure stall information - some tasks waiting, 10 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.memorySomeAvg60, err = h.meter.Float64ObservableGauge(
		"system.psi.memory.some.avg60",
		metric.WithDescription("Memory pressure stall information - some tasks waiting, 60 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.memorySomeAvg300, err = h.meter.Float64ObservableGauge(
		"system.psi.memory.some.avg300",
		metric.WithDescription("Memory pressure stall information - some tasks waiting, 300 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.memorySomeTotal, err = h.meter.Int64ObservableCounter(
		"system.psi.memory.some.total",
		metric.WithDescription("Memory pressure stall information - some tasks waiting, total time in microseconds"),
		metric.WithUnit("us"),
	)
	if err != nil {
		return nil, err
	}

	// Memory PSI metrics - full
	pm.memoryFullAvg10, err = h.meter.Float64ObservableGauge(
		"system.psi.memory.full.avg10",
		metric.WithDescription("Memory pressure stall information - all tasks waiting, 10 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.memoryFullAvg60, err = h.meter.Float64ObservableGauge(
		"system.psi.memory.full.avg60",
		metric.WithDescription("Memory pressure stall information - all tasks waiting, 60 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.memoryFullAvg300, err = h.meter.Float64ObservableGauge(
		"system.psi.memory.full.avg300",
		metric.WithDescription("Memory pressure stall information - all tasks waiting, 300 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.memoryFullTotal, err = h.meter.Int64ObservableCounter(
		"system.psi.memory.full.total",
		metric.WithDescription("Memory pressure stall information - all tasks waiting, total time in microseconds"),
		metric.WithUnit("us"),
	)
	if err != nil {
		return nil, err
	}

	// IO PSI metrics - some
	pm.ioSomeAvg10, err = h.meter.Float64ObservableGauge(
		"system.psi.io.some.avg10",
		metric.WithDescription("IO pressure stall information - some tasks waiting, 10 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.ioSomeAvg60, err = h.meter.Float64ObservableGauge(
		"system.psi.io.some.avg60",
		metric.WithDescription("IO pressure stall information - some tasks waiting, 60 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.ioSomeAvg300, err = h.meter.Float64ObservableGauge(
		"system.psi.io.some.avg300",
		metric.WithDescription("IO pressure stall information - some tasks waiting, 300 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.ioSomeTotal, err = h.meter.Int64ObservableCounter(
		"system.psi.io.some.total",
		metric.WithDescription("IO pressure stall information - some tasks waiting, total time in microseconds"),
		metric.WithUnit("us"),
	)
	if err != nil {
		return nil, err
	}

	// IO PSI metrics - full
	pm.ioFullAvg10, err = h.meter.Float64ObservableGauge(
		"system.psi.io.full.avg10",
		metric.WithDescription("IO pressure stall information - all tasks waiting, 10 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.ioFullAvg60, err = h.meter.Float64ObservableGauge(
		"system.psi.io.full.avg60",
		metric.WithDescription("IO pressure stall information - all tasks waiting, 60 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.ioFullAvg300, err = h.meter.Float64ObservableGauge(
		"system.psi.io.full.avg300",
		metric.WithDescription("IO pressure stall information - all tasks waiting, 300 second average"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pm.ioFullTotal, err = h.meter.Int64ObservableCounter(
		"system.psi.io.full.total",
		metric.WithDescription("IO pressure stall information - all tasks waiting, total time in microseconds"),
		metric.WithUnit("us"),
	)
	if err != nil {
		return nil, err
	}

	// Register callback for PSI metrics
	_, err = h.meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			return pm.observePSI(ctx, o)
		},
		pm.cpuSomeAvg10,
		pm.cpuSomeAvg60,
		pm.cpuSomeAvg300,
		pm.cpuSomeTotal,
		pm.memorySomeAvg10,
		pm.memorySomeAvg60,
		pm.memorySomeAvg300,
		pm.memorySomeTotal,
		pm.memoryFullAvg10,
		pm.memoryFullAvg60,
		pm.memoryFullAvg300,
		pm.memoryFullTotal,
		pm.ioSomeAvg10,
		pm.ioSomeAvg60,
		pm.ioSomeAvg300,
		pm.ioSomeTotal,
		pm.ioFullAvg10,
		pm.ioFullAvg60,
		pm.ioFullAvg300,
		pm.ioFullTotal,
	)
	if err != nil {
		return nil, err
	}

	return pm, nil
}

// observePSI reads PSI metrics and records observations
func (pm *psiMetrics) observePSI(ctx context.Context, o metric.Observer) error {
	cpuStats, err := readPSIFile(psiCPUFile)
	if err == nil {
		o.ObserveFloat64(pm.cpuSomeAvg10, cpuStats.some.avg10)
		o.ObserveFloat64(pm.cpuSomeAvg60, cpuStats.some.avg60)
		o.ObserveFloat64(pm.cpuSomeAvg300, cpuStats.some.avg300)
		o.ObserveInt64(pm.cpuSomeTotal, cpuStats.some.total)
	}

	memStats, err := readPSIFile(psiMemoryFile)
	if err == nil {
		o.ObserveFloat64(pm.memorySomeAvg10, memStats.some.avg10)
		o.ObserveFloat64(pm.memorySomeAvg60, memStats.some.avg60)
		o.ObserveFloat64(pm.memorySomeAvg300, memStats.some.avg300)
		o.ObserveInt64(pm.memorySomeTotal, memStats.some.total)
		o.ObserveFloat64(pm.memoryFullAvg10, memStats.full.avg10)
		o.ObserveFloat64(pm.memoryFullAvg60, memStats.full.avg60)
		o.ObserveFloat64(pm.memoryFullAvg300, memStats.full.avg300)
		o.ObserveInt64(pm.memoryFullTotal, memStats.full.total)
	}

	ioStats, err := readPSIFile(psiIOFile)
	if err == nil {
		o.ObserveFloat64(pm.ioSomeAvg10, ioStats.some.avg10)
		o.ObserveFloat64(pm.ioSomeAvg60, ioStats.some.avg60)
		o.ObserveFloat64(pm.ioSomeAvg300, ioStats.some.avg300)
		o.ObserveInt64(pm.ioSomeTotal, ioStats.some.total)
		o.ObserveFloat64(pm.ioFullAvg10, ioStats.full.avg10)
		o.ObserveFloat64(pm.ioFullAvg60, ioStats.full.avg60)
		o.ObserveFloat64(pm.ioFullAvg300, ioStats.full.avg300)
		o.ObserveInt64(pm.ioFullTotal, ioStats.full.total)
	}

	return nil
}

// readPSIFile reads and parses a PSI file
func readPSIFile(path string) (*psiStats, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read PSI file %s: %w", path, err)
	}

	return parsePSI(string(content))
}

// parsePSI parses PSI file content
// Format:
// some avg10=0.00 avg60=0.00 avg300=0.00 total=0
// full avg10=0.00 avg60=0.00 avg300=0.00 total=0
func parsePSI(content string) (*psiStats, error) {
	stats := &psiStats{}
	lines := strings.Split(strings.TrimSpace(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid PSI line format: %s", line)
		}

		pressureType := parts[0]
		var avg10, avg60, avg300 float64
		var total int64
		var err error

		// Parse avg10=X.XX
		if strings.HasPrefix(parts[1], "avg10=") {
			avg10, err = strconv.ParseFloat(strings.TrimPrefix(parts[1], "avg10="), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse avg10: %w", err)
			}
		}

		// Parse avg60=X.XX
		if strings.HasPrefix(parts[2], "avg60=") {
			avg60, err = strconv.ParseFloat(strings.TrimPrefix(parts[2], "avg60="), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse avg60: %w", err)
			}
		}

		// Parse avg300=X.XX
		if strings.HasPrefix(parts[3], "avg300=") {
			avg300, err = strconv.ParseFloat(strings.TrimPrefix(parts[3], "avg300="), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse avg300: %w", err)
			}
		}

		// Parse total=XXXXX
		if strings.HasPrefix(parts[4], "total=") {
			total, err = strconv.ParseInt(strings.TrimPrefix(parts[4], "total="), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse total: %w", err)
			}
		}

		switch pressureType {
		case "some":
			stats.some.avg10 = avg10
			stats.some.avg60 = avg60
			stats.some.avg300 = avg300
			stats.some.total = total
		case "full":
			stats.full.avg10 = avg10
			stats.full.avg60 = avg60
			stats.full.avg300 = avg300
			stats.full.total = total
		default:
			return nil, fmt.Errorf("unknown pressure type: %s", pressureType)
		}
	}

	return stats, nil
}

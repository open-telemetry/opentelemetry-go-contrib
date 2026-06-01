// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package host provides the conventional host metric instruments
// specified by OpenTelemetry.  Host metric events are sometimes
// collected through the OpenTelemetry Collector "hostmetrics"
// receiver running as an agent; this instrumentation is an
// alternative for processes that want to record the same information
// without an agent.
//
// The metric events produced are listed here with attribute dimensions.
//
//	Name                              Attribute
//
// ----------------------------------------------------------------------
//
//	process.cpu.time                  state=user|system
//	system.cpu.time                   state=user|system|other|idle
//	system.memory.usage               state=used|available
//	system.memory.utilization         state=used|available
//	system.network.io                 direction=transmit|receive
//
// Linux-specific Pressure Stall Information (PSI) metrics:
//
//	system.psi.cpu.some.avg10         (no attributes)
//	system.psi.cpu.some.avg60         (no attributes)
//	system.psi.cpu.some.avg300        (no attributes)
//	system.psi.cpu.some.total         (no attributes)
//	system.psi.memory.some.avg10      (no attributes)
//	system.psi.memory.some.avg60      (no attributes)
//	system.psi.memory.some.avg300     (no attributes)
//	system.psi.memory.some.total      (no attributes)
//	system.psi.memory.full.avg10      (no attributes)
//	system.psi.memory.full.avg60      (no attributes)
//	system.psi.memory.full.avg300     (no attributes)
//	system.psi.memory.full.total      (no attributes)
//	system.psi.io.some.avg10          (no attributes)
//	system.psi.io.some.avg60          (no attributes)
//	system.psi.io.some.avg300         (no attributes)
//	system.psi.io.some.total          (no attributes)
//	system.psi.io.full.avg10          (no attributes)
//	system.psi.io.full.avg60          (no attributes)
//	system.psi.io.full.avg300         (no attributes)
//	system.psi.io.full.total          (no attributes)
//
// PSI metrics are only available on Linux systems with kernel 4.20+.
// "some" indicates that some tasks are stalled, "full" indicates all tasks are stalled.
// The avg* metrics represent pressure averages over 10, 60, and 300 second windows.
// The total metrics represent cumulative stall time in microseconds.
//
// See https://github.com/open-telemetry/oteps/blob/main/text/0119-standard-system-metrics.md
// for the definition of these metric instruments.
// For PSI metrics, see https://docs.kernel.org/accounting/psi.html
package host // import "go.opentelemetry.io/contrib/instrumentation/host"

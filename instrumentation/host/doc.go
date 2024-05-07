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
//	Name			Attribute
//
// ----------------------------------------------------------------------
//
//	process.cpu.time           state=user|system
//	system.cpu.time            state=user|system|other|idle
//	system.memory.usage        state=used|available
//	system.memory.utilization  state=used|available
//	system.network.io          direction=transmit|receive
//
// See https://github.com/open-telemetry/oteps/blob/main/text/0119-standard-system-metrics.md
// for the definition of these metric instruments.
package host // import "go.opentelemetry.io/contrib/instrumentation/host"

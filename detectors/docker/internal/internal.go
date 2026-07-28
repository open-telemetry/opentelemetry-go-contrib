// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package internal provides helpers for mapping Go runtime OS/arch identifiers
// to OpenTelemetry semantic convention values.
package internal

// GOOSToOSType maps a runtime.GOOS-like value to os.type style.
func GOOSToOSType(goos string) string {
	if goos == "dragonfly" {
		return "dragonflybsd"
	}
	return goos
}

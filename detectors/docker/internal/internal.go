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

// GOARCHToHostArch maps a runtime.GOARCH-like value to host.arch style.
func GOARCHToHostArch(goarch string) string {
	// These cases differ from the spec well-known values
	switch goarch {
	case "arm":
		return "arm32"
	case "ppc64le":
		return "ppc64"
	case "386":
		return "x86"
	}

	// Other cases either match the spec or are not well-known (so we use a custom value)
	return goarch
}

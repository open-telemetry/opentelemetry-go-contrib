// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// +build !darwin
// +build !dragonfly
// +build !freebsd
// +build !linux
// +build !netbsd
// +build !openbsd
// +build !solaris
// +build !windows

package host // import "go.opentelemetry.io/contrib/detectors/host"

// hostIDReaderUnsupported is a placeholder implementation for operating systems
// for which this project currently doesn't support host.id
// attribute detection. See build tags declaration early on this file
// for a list of unsupported OSes.
func getHostId() (string, error) {
	return "<unknown>", nil
}

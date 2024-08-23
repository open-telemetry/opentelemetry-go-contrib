// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmacaron // import "go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron"

// Version is the current release version of the macron instrumentation.
func Version() string {
	return "0.54.0"
	// This string is updated by the pre_release.sh script during release
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
//
// Deprecated: Use [Version] instead.
func SemVersion() string {
	return Version()
}

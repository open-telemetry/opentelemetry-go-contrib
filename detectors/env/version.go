// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package env // import "go.opentelemetry.io/contrib/detectors/env"

// Version is the current release version of the env detector.
func Version() string {
	return "0.1.0"
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
//
// Deprecated: Use [Version] instead.
func SemVersion() string {
	return Version()
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package aws contains OpenTelemetry propagators that use AWS propagation
// formats.
package aws // import "go.opentelemetry.io/contrib/propagators/aws"

// Version is the current release version of the AWS XRay propagator.
func Version() string {
	return "1.40.0"
	// This string is updated by the pre_release.sh script during release
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
//
// Deprecated: Use [Version] instead.
func SemVersion() string {
	return Version()
}

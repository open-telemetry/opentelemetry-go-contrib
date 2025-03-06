// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgin // import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

// Version is the current release version of the gin instrumentation.
func Version() string {
	return "0.60.0"
	// This string is updated by the pre_release.sh script during release
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
//
// Deprecated: Use [Version] instead.
func SemVersion() string {
	return Version()
}

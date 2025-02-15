// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/test"

// Version is the current release version of the mongo-driver instrumentation test module.
func Version() string {
	return "0.59.0"
	// This string is updated by the pre_release.sh script during release
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
//
// Deprecated: Use [Version] instead.
func SemVersion() string {
	return Version()
}

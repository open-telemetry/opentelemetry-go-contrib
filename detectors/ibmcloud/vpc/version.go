// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpc // import "go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"

// Version is the current release version of the IBM Cloud VPC resource detector.
func Version() string {
	return "0.15.0"
	// This string is updated by the pre_release.sh script during release
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
//
// Deprecated: Use [Version] instead.
func SemVersion() string {
	return Version()
}

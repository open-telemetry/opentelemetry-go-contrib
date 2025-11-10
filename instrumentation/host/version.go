// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package host // import "go.opentelemetry.io/contrib/instrumentation/host"

// Version is the current release version of the host instrumentation.
func Version() string {
	return "0.63.0"
	// This string is updated by the pre_release.sh script during release
}

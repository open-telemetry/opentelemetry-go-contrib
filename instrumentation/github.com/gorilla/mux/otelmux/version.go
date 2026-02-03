// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

// Version is the current release version of the gorilla/mux instrumentation.
func Version() string {
	return "0.65.0"
	// This string is updated by the pre_release.sh script during release
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

// Version is the current release version of the mongo-driver instrumentation.
func Version() string {
	return "0.64.0"
	// This string is updated by the pre_release.sh script during release
}

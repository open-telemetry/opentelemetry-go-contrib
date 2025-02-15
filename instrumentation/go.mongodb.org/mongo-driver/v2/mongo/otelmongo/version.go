// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

// Version is the current release version of the mongo-go-driver V2 instrumentation.
func Version() string {
	return "0.60.0"
	// This string is updated by the pre_release.sh script during release
}

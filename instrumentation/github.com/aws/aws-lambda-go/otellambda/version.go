// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"

// Version is the current release version of the AWS Lambda instrumentation.
func Version() string {
	return "0.64.0"
	// This string is updated by the pre_release.sh script during release
}
